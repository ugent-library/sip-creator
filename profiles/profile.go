package profiles

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
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

type Config struct {
	Source      string
	Destination string
	Logger      *slog.Logger
}

type Profile struct {
	BaseDir string
	InDir   string
	Logger  *slog.Logger
}

func New(config *Config) *Profile {
	return &Profile{
		BaseDir: config.Destination,
		InDir:   config.Source,
		Logger:  config.Logger,
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

func (p *Profile) createIntellectualEntity() *structure.Entity {
	return structure.NewEntity()
}

type Description interface {
	GetLocalIdentifier(scheme string) string
	SetObjectIdentifier(id string)
}

func (p *Profile) createDescriptiveFile(src, dir string, fn func(d Description)) *structure.File {
	fi, err := os.Lstat(src)
	if err != nil {
		panic(err)
	}

	if fi.IsDir() {
		panic(fmt.Sprintf("%s is a directory, not a metadata file.", src))
	}

	base := path.Base(src)
	ext := path.Ext(base)
	name := base[0:len(base)-len(ext)] + ".xml"

	dest := fmt.Sprintf("%s/%s", dir, name)

	f, err := os.Open(src)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	d := metadata.Decode(f)

	// Per the spec, we want to swap in the premis identifier for dcterms:identifier,
	//   and swap out existing dcterms:identifier values from the source, keeping them with
	//   the entity when we generate the premis file. Do this in a callback.
	fn(d)

	file := createMetadataFile(dest, func(w io.Writer) error {
		return metadata.Encode(w, d)
	})

	file.Path = fmt.Sprintf(".%s", dest[len(p.BaseDir):])

	return file
}

func (p *Profile) createRepresentation(label string) *structure.Representation {
	createDir(fmt.Sprintf("%s/representations/%s/data", p.BaseDir, label))
	createDir(fmt.Sprintf("%s/representations/%s/metadata/preservation", p.BaseDir, label))

	r := structure.NewRepresentation(label)

	// TODO generate an optional description file (dc+schema, dc, mods) for representations
	//   e.g. when licensing overrides the licensing of the associated entity

	return r
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

		r := p.createRepresentation(label)

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

		base := fmt.Sprintf("%s/representations/%s", p.BaseDir, label)
		dest := fmt.Sprintf("%s/data/%s", base, path.Base(src))
		copy(src, dest)

		// TODO make this a registrable identificator, add support for Droid as well
		formatter := siegfried.New("sf", []string{"-hash", "md5", "-json"})
		f := formatter.Process(dest)

		f.Path = fmt.Sprintf(".%s", dest[len(base):])

		fn(f)

		return nil
	})

	if err != nil {
		panic(err)
	}
}

func (p *Profile) generatePackagePremis(e *structure.Entity) *structure.File {
	// TODO also account for sub-iE(s) tied to the root entity
	path := fmt.Sprintf("%s/metadata/preservation/premis.xml", p.BaseDir)
	f := createMetadataFile(path, func(w io.Writer) error {
		return premis.EncodeEntity(w, e)
	})

	// Set the relative path within the package for METS
	f.Path = fmt.Sprintf(".%s", path[len(p.BaseDir):])

	return f
}

func (p *Profile) generateRepresentationPremis(r *structure.Representation) *structure.File {
	// path := fmt.Sprintf("%s/representations/%s/metadata/preservation/premis.xml", p.BaseDir, r.Label)

	base := fmt.Sprintf("%s/representations/%s", p.BaseDir, r.Label)
	path := fmt.Sprintf("%s/metadata/preservation/premis.xml", base)

	f := createMetadataFile(path, func(w io.Writer) error {
		return premis.EncodeRepresentation(w, r)
	})

	f.Path = fmt.Sprintf(".%s", path[len(base):])

	return f
}

func (p *Profile) generatePackageMets(pkg *structure.Package) *structure.File {
	path := fmt.Sprintf("%s/mets.xml", p.BaseDir)
	f := createMetadataFile(path, func(w io.Writer) error {
		return mets.EncodePackage(w, pkg)
	})

	f.Path = fmt.Sprintf(".%s", path[len(p.BaseDir):])

	return f
}

func (p *Profile) generateRepresentationMets(r *structure.Representation) *structure.File {
	path := fmt.Sprintf("%s/representations/%s/mets.xml", p.BaseDir, r.Label)
	f := createMetadataFile(path, func(w io.Writer) error {
		return mets.EncodeRepresentation(w, r)
	})

	f.Path = fmt.Sprintf(".%s", path[len(p.BaseDir):])

	return f
}

func createMetadataFile(dest string, fn func(w io.Writer) error) *structure.File {
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
	file.Name = path.Base(dest)
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
	err := os.MkdirAll(path, 0775)
	if err != nil {
		panic(err)
	}
}
