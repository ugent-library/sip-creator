package structure

type SiegfriedFile struct {
	Files []*SFile `json:"files"`
}

type SFile struct {
	Filename string    `json:"filename"`
	Filesize int       `json:"filesize"`
	Modified string    `json:"modified"`
	Errors   string    `json:"errors"`
	Matches  []*SMatch `json:"matches"`
	Checksum string    `json:"md5"`
}

func (sf *SiegfriedFile) Find(filename string) *SFile {
	for _, sf := range sf.Files {
		if sf.Filename == filename {
			return sf
		}
	}

	return nil
}

type SMatch struct {
	NS      string `json:"ns"`
	ID      string `json:"id"`
	Format  string `json:"format"`
	Version string `json:"version"`
	Mime    string `json:"mime"`
	Class   string `json:"class"`
	Basis   string `json:"basis"`
	Warning string `json:"warning"`
}
