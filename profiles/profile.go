package profiles

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/ugent-library/sip-creator/encoders/metadata"
	"github.com/ugent-library/sip-creator/encoders/mets"
	"github.com/ugent-library/sip-creator/encoders/premis"
	"github.com/ugent-library/sip-creator/siegfried"
	"github.com/ugent-library/sip-creator/structure"
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

func (p *Profile) createIntellectualEntity(src string) *structure.Entity {
	entity := structure.NewEntity()

	f, err := os.Lstat(src)
	if err != nil {
		panic(err)
	}

	if f.IsDir() {
		panic(fmt.Sprintf("%s is a directory, not a metadata file.", src))
	}

	// TODO split this out as a helper function, representations can have
	//   descriptive files as well (optional)
	base := path.Base(src)
	ext := path.Ext(base)
	name := base[0:len(base)-len(ext)] + ".xml"

	dest := fmt.Sprintf("%s/metadata/descriptive/%s", p.BaseDir, name)

	file := createMetadataFile(dest, func(w io.Writer) error {
		// TODO Per the spec, we want to swap in the premis identifier for dcterms:identifier,
		//   and swap out existing dcterms:identifier values from the source, keeping them with
		//   the entity when we generate the premis file
		description := metadata.Decode(src)
		description.Identifier = entity.Identifier
		return metadata.Encode(w, description)
	})

	entity.AddDescriptionFile(file)

	return entity
}

func (p *Profile) eachDirectory(fn func(dir string, r *structure.Representation)) {
	err := filepath.Walk(p.InDir, func(dir string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			return nil
		}

		// TODO fix case "representation_0"
		label := path.Base(dir)
		rx, _ := regexp.Compile("representation_([0-9]+)$")
		if !rx.MatchString(label) {
			return nil
		}

		createDir(fmt.Sprintf("%s/representations/%s/data", p.BaseDir, label))
		createDir(fmt.Sprintf("%s/representations/%s/metadata/preservation", p.BaseDir, label))

		r := structure.NewRepresentation(label)

		fn(dir, r)

		return nil
	})

	if err != nil {
		panic(err)
	}
}

func (p *Profile) eachFile(dir, label string, fn func(r *structure.File)) {
	err := filepath.Walk(dir, func(src string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		// TODO ignore all descriptive files per the spec (dc, dc+schema, mods)

		dest := fmt.Sprintf("%s/representations/%s/data/%s", p.BaseDir, label, path.Base(src))
		copy(src, dest)

		// TODO make this a registrable identificator, add support for Droid as well
		formatter := siegfried.New("sf", []string{"-hash", "md5", "-json"})
		f := formatter.Process(dest)

		fn(f)

		return nil
	})

	if err != nil {
		panic(err)
	}
}

func (p *Profile) createPackage() *structure.Package {
	// Create skeleton
	packageDirs := []string{
		fmt.Sprintf("%s/metadata/descriptive", p.BaseDir),
		fmt.Sprintf("%s/metadata/preservation", p.BaseDir),
		fmt.Sprintf("%s/representations", p.BaseDir),
	}

	for _, pd := range packageDirs {
		createDir(pd)
	}

	// Create a new package
	return structure.NewPackage()
}

func (p *Profile) createPremisPackage(path string, pkg *structure.Package, root *structure.Entity) {
	file := createMetadataFile(path, func(w io.Writer) error {
		return premis.EncodeEntity(w, root)
	})

	pkg.AddPremisFile(file)
}

func (p *Profile) createPremisRepresentation(path string, representation *structure.Representation, entity structure.Entity) {
	file := createMetadataFile(path, func(w io.Writer) error {
		return premis.EncodeRepresentation(w, entity, representation)
	})

	representation.AddPremisFile(file)
}

func (p *Profile) createMetsPackage(path string, pkg *structure.Package) {
	createMetadataFile(path, func(w io.Writer) error {
		return mets.EncodePackage(w, pkg)
	})
}

func (p *Profile) createMetsRepresentation(path string, representation *structure.Representation) {
	file := createMetadataFile(path, func(w io.Writer) error {
		return mets.EncodeRepresentation(w, representation)
	})

	representation.AddMetsFile(file)
}

// func (p *Profile) createDescriptionFile(path string, entity structure.Entity) {
// 	file := createMetadataFile(path, func(w io.Writer) error {
// 		return metadata.EncodeMetadata(w, entity)
// 	})

// 	entity.AddDescriptionFile(file)
// }

type encoder func(w io.Writer) error

func createMetadataFile(dest string, fn encoder) *structure.File {
	hash := md5.New()
	var buf1, buf2 bytes.Buffer
	w := io.MultiWriter(&buf1, &buf2)

	if err := fn(w); err != nil {
		panic(err)
	}

	if _, err := io.Copy(hash, &buf1); err != nil {
		panic(err)
	}

	of, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
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

	file := structure.NewFile()
	file.Name = dest
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
