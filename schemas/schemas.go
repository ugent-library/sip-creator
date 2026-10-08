// Package schemas bundles every XSD that the profiles can ship in a
// package's schemas/ directory.
package schemas

import (
	"embed"
	"io/fs"
)

//go:embed *.xsd
var fsys embed.FS

// Get returns the bundled XSDs, keyed by file name.
func Get() map[string][]byte {
	files := make(map[string][]byte)

	fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, _ error) error {
		if d.IsDir() {
			return nil
		}

		buf, err := fsys.ReadFile(name)
		if err != nil {
			// Reading a file that the embedded file system itself listed
			// cannot fail at runtime, so a failure is a programmer error.
			panic(err)
		}

		files[name] = buf

		return nil
	})

	return files
}
