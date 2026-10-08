package ugent

// dcElements is the Dublin Core Metadata Element Set (ISO 15836), in the
// order DCMI lists it. In Simple Dublin Core a term's key is the element
// name itself. Every element is optional and repeatable in the standard.
var dcElements = []string{
	"title", "creator", "subject", "description", "publisher",
	"contributor", "date", "type", "format", "identifier",
	"source", "language", "relation", "coverage", "rights",
}

// dcRequired lists the keys a package-level description must state:
// docs/profiles/ugent-basic.md §3, "The description MUST state an
// identifier and a title."
var dcRequired = []string{"identifier", "title"}

var dcElementSet = func() map[string]bool {
	set := make(map[string]bool, len(dcElements))
	for _, el := range dcElements {
		set[el] = true
	}
	return set
}()
