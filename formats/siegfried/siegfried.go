package siegfried

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path"
	"strconv"

	"github.com/ugent-library/sip-creator/formats"
	"github.com/ugent-library/sip-creator/sip"
)

func init() {
	formats.Register("siegfried", New)
}

func New(bin string, args []string) (formats.Identificator, error) {
	return &siegfried{
		bin:  bin,
		args: args,
	}, nil
}

type siegfried struct {
	bin  string
	args []string
}

type Output struct {
	Files []*File `json:"files"`
}

func (o *Output) FirstFile() *File {
	if len(o.Files) > 0 {
		return o.Files[0]
	}

	return nil
}

type File struct {
	Filename string   `json:"filename"`
	Filesize int      `json:"filesize"`
	Modified string   `json:"modified"`
	Errors   string   `json:"errors"`
	Matches  []*Match `json:"matches"`
	Checksum string   `json:"md5"`
}

func (f *File) FirstMatch() *Match {
	if len(f.Matches) > 0 {
		return f.Matches[0]
	}

	return nil
}

type Match struct {
	NS      string `json:"ns"`
	ID      string `json:"id"`
	Format  string `json:"format"`
	Version string `json:"version"`
	Mime    string `json:"mime"`
	Class   string `json:"class"`
	Basis   string `json:"basis"`
	Warning string `json:"warning"`
}

func (s *siegfried) Process(f string) *sip.File {
	var b bytes.Buffer

	s.args = append(s.args, f)

	// TODO handle errors if the command fails (e.g. not installed)
	bin := exec.Command(s.bin, s.args...)
	bin.Stdout = &b
	_ = bin.Run()

	var out *Output

	if err := json.Unmarshal(b.Bytes(), &out); err != nil {
		panic(err)
	}

	file := sip.NewFile()
	file.Name = path.Base(f)

	if sfile := out.FirstFile(); sfile != nil {
		if match := sfile.FirstMatch(); match != nil {
			fr := sip.NewFormatRegistry()
			fr.Key = match.ID
			fr.Name = match.NS

			file.Format = &sip.Format{
				FormatRegistry: fr,
			}
			file.Checksum = sfile.Checksum
			file.Size = strconv.Itoa(sfile.Filesize)
			file.Created = sfile.Modified
		}
	}

	return file
}
