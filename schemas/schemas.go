package schemas

import (
	"embed"
	"io/fs"
)

//go:embed *.xsd
var fsys embed.FS

func Get() map[string][]byte {
	files := make(map[string][]byte)

	fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, _ error) error {
		if d.IsDir() {
			return nil
		}

		buf, err := fsys.ReadFile(name)
		if err != nil {
			panic(err)
		}

		files[name] = buf

		return nil
	})

	return files
}
