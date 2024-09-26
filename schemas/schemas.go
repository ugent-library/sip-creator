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

// func Create(dir string) {
// 	fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, _ error) error {
// 		if d.IsDir() {
// 			return nil
// 		}

// 		buf, err := fsys.ReadFile(name)
// 		if err != nil {
// 			panic(err)
// 		}

// 		dest := fmt.Sprintf("%s/%s", dir, name)
// 		of, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
// 		if err != nil {
// 			panic(err)
// 		}
// 		defer of.Close()

// 		if _, err := of.Write(buf); err != nil {
// 			panic(err)
// 		}

// 		return nil
// 	})
// }
