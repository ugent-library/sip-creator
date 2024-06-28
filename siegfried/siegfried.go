package siegfried

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path"
	"strconv"

	"github.com/ugent-library/sip-creator/structure"
)

type siegfried struct {
	cmd  string
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

func New(cmd string, args []string) *siegfried {
	return &siegfried{
		cmd:  cmd,
		args: args,
	}
}

func (s *siegfried) Process(f string) *structure.File {
	var b bytes.Buffer

	s.args = append(s.args, f)

	cmd := exec.Command(s.cmd, s.args...)
	cmd.Stdout = &b
	_ = cmd.Run()

	var out *Output

	if err := json.Unmarshal(b.Bytes(), &out); err != nil {
		panic(err)
	}

	file := structure.NewFile()
	file.Name = path.Base(f)

	if sfile := out.FirstFile(); sfile != nil {
		if match := sfile.FirstMatch(); match != nil {
			fr := structure.NewFormatRegistry()
			fr.Key = match.ID
			fr.Name = match.NS

			file.Format = &structure.Format{
				FormatRegistry: fr,
			}
			file.Checksum = sfile.Checksum
			file.Size = strconv.Itoa(sfile.Filesize)
			file.Created = sfile.Modified
		}
	}

	return file
}
