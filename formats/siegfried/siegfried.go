package siegfried

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"slices"

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
	Errors   string   `json:"errors"`
	Matches  []*Match `json:"matches"`
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

func (s *siegfried) Identify(path string) (*sip.Format, error) {
	// A fresh slice per call: appending to s.args would accumulate paths
	// across calls, running sf against every file seen so far.
	args := append(slices.Clone(s.args), path)

	var out bytes.Buffer
	cmd := exec.Command(s.bin, args...)
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("format identification (%s): %w", s.bin, err)
	}

	var result *Output
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("format identification: parse %s output: %w", s.bin, err)
	}

	sfile := result.FirstFile()
	if sfile == nil {
		return nil, nil
	}
	if sfile.Errors != "" {
		return nil, fmt.Errorf("format identification (%s): %s", s.bin, sfile.Errors)
	}

	match := sfile.FirstMatch()
	if match == nil {
		return nil, nil
	}

	fr := sip.NewFormatRegistry()
	fr.Name = match.NS
	fr.Key = match.ID

	return &sip.Format{FormatRegistry: fr}, nil
}
