package input

import "testing"

func TestIsOSArtifact(t *testing.T) {
	for _, name := range []string{".DS_Store", "Thumbs.db", "thumbs.db", "desktop.ini", "._resource"} {
		if !isOSArtifact(name) {
			t.Errorf("%q must be ignored as an OS artifact", name)
		}
	}
	for _, name := range []string{"scan.tiff", ".gitignore", "_underscore.txt"} {
		if isOSArtifact(name) {
			t.Errorf("%q must not be treated as an OS artifact", name)
		}
	}
}
