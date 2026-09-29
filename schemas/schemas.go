// Package schemas bundles the XSDs a SIP can carry in its schemas/ dir: the
// union of what the profiles ship. Which of them one package ships follows
// from its documents, so no package carries the whole bundle.
package schemas

import (
	"embed"
	"io/fs"
)

//go:embed *.xsd
var fsys embed.FS

// Get returns the bundled XSDs by filename.
func Get() map[string][]byte {
	files := make(map[string][]byte)

	fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, _ error) error {
		if d.IsDir() {
			return nil
		}

		buf, err := fsys.ReadFile(name)
		if err != nil {
			// Reading a file the embed FS itself listed cannot fail at
			// runtime; a failure is a programmer error.
			panic(err)
		}

		files[name] = buf

		return nil
	})

	return files
}
