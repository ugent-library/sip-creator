package profiles

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/samber/lo"
	"github.com/ugent-library/sip-creator/encoders/dc"
	"github.com/ugent-library/sip-creator/encoders/mets"
	"github.com/ugent-library/sip-creator/encoders/premis"
	"github.com/ugent-library/sip-creator/metadata"
)

type Profile struct {
	BaseDir string
	InDir   string
}

func New(src, dest string) *Profile {
	return &Profile{
		BaseDir: dest,
		InDir:   src,
	}
}

func (p *Profile) createIntellectualEntity(src, label string) metadata.Entity {
	// Create a new intellectual entity
	entity := metadata.NewDublinCoreEntity(label)

	// Parse metadata file
	mf, err := os.Open(src)
	if err != nil {
		panic(err)
	}
	defer mf.Close()

	// Add the dublin core description
	bts, _ := io.ReadAll(mf)
	entity.AddDescription(bts)

	return entity
}

func (p *Profile) createRepresentation(src, label string) *metadata.Representation {
	representation := metadata.NewRepresentation(label)

	createDir(fmt.Sprintf("%s/representations/%s/data", p.BaseDir, label))
	createDir(fmt.Sprintf("%s/representations/%s/metadata/preservation", p.BaseDir, label))

	// TODO Look for description files, create corresponding entities, hook 'em to the representation

	// Load Siegfried file
	// TODO move file characterisation into an abstraction
	sf, err := os.Open(fmt.Sprintf("%s/siegfried.json", p.InDir))
	if err != nil {
		panic(err)
	}
	defer sf.Close()

	var siegfried metadata.SiegfriedFile
	bts, _ := io.ReadAll(sf)
	json.Unmarshal(bts, &siegfried)

	// // Copy essence files
	err = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		if !lo.Contains([]string{"dc+schema.json", "dc.json", "mods.json"}, info.Name()) {
			file := metadata.NewFile()

			// Fetch the PRONOM registry key & MD5 checksum from the siegfried output
			sfile := siegfried.Find(fmt.Sprintf("%s/%s", label, info.Name()))
			if sfile != nil {
				file.Format = sfile.Matches[0].Format
				file.Checksum = sfile.Checksum
				file.Name = info.Name()
				file.Size = strconv.Itoa(sfile.Filesize)
				file.Created = sfile.Modified
			}

			// Copy essence
			// TODO handling of paths
			copy(fmt.Sprintf("%s/%s", src, file.Name), fmt.Sprintf("%s/representations/%s/data/%s", p.BaseDir, label, file.Name))

			representation.AddFile(file)
		}

		return nil
	})

	if err != nil {
		panic(err)
	}

	return representation
}

func (p *Profile) createPremisPackage(path string, pkg *metadata.Package, root metadata.Entity) {
	file := createMetadataFile(path, func(w io.Writer) error {
		return premis.EncodeEntity(w, root)
	})

	pkg.AddPremisFile(file)
}

func (p *Profile) createPremisRepresentation(path string, representation *metadata.Representation, entity metadata.Entity) {
	file := createMetadataFile(path, func(w io.Writer) error {
		return premis.EncodeRepresentation(w, entity, representation)
	})

	representation.AddPremisFile(file)
}

func (p *Profile) createMetsPackage(path string, pkg *metadata.Package) {
	createMetadataFile(path, func(w io.Writer) error {
		return mets.EncodePackage(w, pkg)
	})
}

func (p *Profile) createMetsRepresentation(path string, representation *metadata.Representation) {
	file := createMetadataFile(path, func(w io.Writer) error {
		return mets.EncodeRepresentation(w, representation)
	})

	representation.AddMetsFile(file)
}

func (p *Profile) createDescriptionFile(path string, pkg *metadata.Package, entity metadata.Entity) {
	file := createMetadataFile(path, func(w io.Writer) error {
		return dc.EncodeDescriptive(w, entity)
	})
	pkg.AddDescriptiveFile(file)
}

type encoder func(w io.Writer) error

func createMetadataFile(path string, fn encoder) *metadata.File {
	hash := md5.New()
	var buf1, buf2 bytes.Buffer
	w := io.MultiWriter(&buf1, &buf2)

	if err := fn(w); err != nil {
		panic(err)
	}

	if _, err := io.Copy(hash, &buf1); err != nil {
		panic(err)
	}

	of, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		panic(err)
	}
	defer of.Close()

	buf := make([]byte, 2048)
	for {
		n, err := buf2.Read(buf)
		if err != nil && err != io.EOF {
			panic(err)
		}
		if n == 0 {
			break
		}

		if _, err := of.Write(buf[:n]); err != nil {
			panic(err)
		}
	}

	info, err := of.Stat()
	if err != nil {
		panic(err)
	}

	file := metadata.NewFile()
	file.Name = path
	file.Size = strconv.Itoa(int(info.Size()))
	file.Checksum = hex.EncodeToString(hash.Sum(nil))
	file.Created = info.ModTime().Format(time.RFC3339Nano)

	return file
}

func copy(src, dest string) {
	ipf, err := os.Open(src)
	if err != nil {
		panic(err)
	}

	of, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		panic(err)
	}
	defer of.Close()

	buf := make([]byte, 2048)
	for {
		n, err := ipf.Read(buf)
		if err != nil && err != io.EOF {
			panic(err)
		}
		if n == 0 {
			break
		}

		if _, err := of.Write(buf[:n]); err != nil {
			panic(err)
		}
	}
}

func createDir(path string) {
	log.Println(path)
	err := os.MkdirAll(path, 0775)
	if err != nil {
		panic(err)
	}
}
