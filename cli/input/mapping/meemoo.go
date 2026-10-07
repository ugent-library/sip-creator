package mapping

import (
	"fmt"

	"github.com/ugent-library/sip-creator/profiles/meemoo"
	"github.com/ugent-library/sip-creator/sip"
)

// Meemoo is the basic profile's mapping: the keys the input specification
// lists for Meemoo's basic content profile and the element each one
// states. The keys are the input folder's convention, lowercase and
// without prefix; meemoo.Terms is keyed by the elements as Meemoo's
// specification names them. Which elements exist and what a term may say
// is decided by the terms' Validate.
type Meemoo struct{}

// meemooElements maps each key of description.csv under meemoo/basic onto the
// element it states, in the input specification's table order.
var meemooElements = map[string]string{
	"identifier":   "dcterms:identifier",
	"title":        "dcterms:title",
	"description":  "dcterms:description",
	"created":      "dcterms:created",
	"alternative":  "dcterms:alternative",
	"abstract":     "dcterms:abstract",
	"creator":      "dcterms:creator",
	"contributor":  "dcterms:contributor",
	"publisher":    "dcterms:publisher",
	"issued":       "dcterms:issued",
	"available":    "dcterms:available",
	"subject":      "dcterms:subject",
	"spatial":      "dcterms:spatial",
	"temporal":     "dcterms:temporal",
	"extent":       "dcterms:extent",
	"language":     "dcterms:language",
	"type":         "dcterms:type",
	"ispartof":     "dcterms:isPartOf",
	"license":      "dcterms:license",
	"rights":       "dcterms:rights",
	"rightsholder": "dcterms:rightsHolder",
	"artmedium":    "schema:artMedium",
	"artform":      "schema:artform",
}

// Map renames each term's key to its element, in order. A term whose key
// is not in the table is reported as a *sip.TermError and kept as written,
// so every term keeps its index and a finding of the terms' Validate still
// names the right row.
func (Meemoo) Map(terms []sip.Term) (sip.Description, []error) {
	out := make(meemoo.Terms, len(terms))
	var errs []error
	for i, t := range terms {
		out[i] = t
		element, ok := meemooElements[t.Key]
		if !ok {
			errs = append(errs, &sip.TermError{Index: i, Err: fmt.Errorf("unknown key %q: not a key of the %s profile; see the supported keys in the input specification", t.Key, meemoo.Definition.Name)})
			continue
		}
		out[i].Key = element
	}
	return out, errs
}
