package dc

import "strings"

// elements is the Dublin Core Metadata Element Set (ISO 15836), in the
// order DCMI lists it. Every element is optional and repeatable in Simple
// Dublin Core; which ones a package must state is profile data.
var elements = []string{
	"title", "creator", "subject", "description", "publisher",
	"contributor", "date", "type", "format", "identifier",
	"source", "language", "relation", "coverage", "rights",
}

var elementSet = func() map[string]bool {
	set := make(map[string]bool, len(elements))
	for _, el := range elements {
		set[el] = true
	}
	return set
}()

// ResolveKey maps a plain key from the CSV rows file onto the element it
// names. In Simple Dublin Core the key is the element name; keys are
// case-insensitive per the convention.
func ResolveKey(key string) (element string, ok bool) {
	element = strings.ToLower(key)
	return element, elementSet[element]
}
