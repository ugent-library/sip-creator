package eark

// elements is the Dublin Core Metadata Element Set (ISO 15836), in the
// order DCMI lists it. In Simple Dublin Core a term's key is the element
// name itself. Every element is optional and repeatable; which ones a
// package must state is profile data.
var elements = []string{
	"title", "creator", "subject", "description", "publisher",
	"contributor", "date", "type", "format", "identifier",
	"source", "language", "relation", "coverage", "rights",
}

// required lists the keys a package-level description must state: the
// identity every package states whatever the profile (input specification
// §3), an identifier, which the eark profile keeps in the document as the
// value operators search by (ADR-0012), and a title, the one name every
// consumer shows. Plain E-ARK requires nothing more. A representation's
// description need not state them.
var required = []string{"identifier", "title"}

var elementSet = func() map[string]bool {
	set := make(map[string]bool, len(elements))
	for _, el := range elements {
		set[el] = true
	}
	return set
}()
