package metadata

import (
	"fmt"
	"regexp"
	"strings"
)

// Term is one descriptive statement: a prefixed element name from a
// supported vocabulary, an optional language tag, and the value.
type Term struct {
	Element string // prefixed element name, e.g. "dcterms:title"
	Lang    string // xml:lang value; empty when unspecified
	Value   string
}

// Terms is an ordered list of descriptive statements; the order the
// producer stated them in is preserved through to the emitted XML.
//
// Terms is plainly constructible by design: any producer builds it
// directly — the CLI's metadata.csv decoder is one, an embedding system
// mapping its own records is another (the input-convention plan: the file
// is one transport, not the API). Whatever the producer, Validate holds
// the rules on what a term may say; Encode refuses invalid terms.
type Terms []Term

// dctermsProperties is the DCMI Metadata Terms property vocabulary — the
// closed list a dcterms: element must name, embedded so a typo like
// dcterms:titel is caught at build time instead of by a downstream XSD.
var dctermsProperties = map[string]bool{
	"abstract": true, "accessRights": true, "accrualMethod": true,
	"accrualPeriodicity": true, "accrualPolicy": true, "alternative": true,
	"audience": true, "available": true, "bibliographicCitation": true,
	"conformsTo": true, "contributor": true, "coverage": true,
	"created": true, "creator": true, "date": true, "dateAccepted": true,
	"dateCopyrighted": true, "dateSubmitted": true, "description": true,
	"educationLevel": true, "extent": true, "format": true,
	"hasFormat": true, "hasPart": true, "hasVersion": true,
	"identifier": true, "instructionalMethod": true, "isFormatOf": true,
	"isPartOf": true, "isReferencedBy": true, "isReplacedBy": true,
	"isRequiredBy": true, "issued": true, "isVersionOf": true,
	"language": true, "license": true, "mediator": true, "medium": true,
	"modified": true, "provenance": true, "publisher": true,
	"references": true, "relation": true, "replaces": true,
	"requires": true, "rights": true, "rightsHolder": true, "source": true,
	"spatial": true, "subject": true, "tableOfContents": true,
	"temporal": true, "title": true, "type": true, "valid": true,
}

var (
	// langRx is a pragmatic language-tag shape (primary subtag plus
	// optional subtags), not full BCP 47 validation.
	langRx = regexp.MustCompile(`^[A-Za-z]{2,3}(-[A-Za-z0-9]{1,8})*$`)
	// schemaPropertyRx bounds schema.org property names by shape only: the
	// vocabulary is open-ended, so unlike dcterms there is no closed list
	// to check against.
	schemaPropertyRx = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*$`)
)

// Validate reports why the term cannot be emitted: an element outside the
// supported vocabularies, a malformed language tag, or an empty value.
// These rules bind every producer, not just the CSV transport.
func (t Term) Validate() error {
	prefix, base, prefixed := strings.Cut(t.Element, ":")
	switch {
	case !prefixed:
		return fmt.Errorf("%q is not a prefixed element name (dcterms: or schema:)", t.Element)
	case prefix == "dcterms":
		if !dctermsProperties[base] {
			return fmt.Errorf("%q is not a Dublin Core term", t.Element)
		}
	case prefix == "schema":
		if !schemaPropertyRx.MatchString(base) {
			return fmt.Errorf("%q is not a schema.org property name", t.Element)
		}
	default:
		return fmt.Errorf("unknown vocabulary prefix in %q — supported: dcterms:, schema:", t.Element)
	}
	if t.Lang != "" && !langRx.MatchString(t.Lang) {
		return fmt.Errorf("%q is not a language tag", t.Lang)
	}
	if strings.TrimSpace(t.Value) == "" {
		return fmt.Errorf("%s has an empty value", t.Element)
	}
	return nil
}

// Validate checks every term plus the one cross-term rule: at most one
// dcterms:identifier — the local identifier is an identity, and two of
// them is an ambiguity no consumer can resolve.
func (t Terms) Validate() error {
	identifiers := 0
	for i, term := range t {
		if err := term.Validate(); err != nil {
			return fmt.Errorf("term %d: %w", i+1, err)
		}
		if term.Element == "dcterms:identifier" {
			identifiers++
		}
	}
	if identifiers > 1 {
		return fmt.Errorf("dcterms:identifier appears %d times — give exactly one", identifiers)
	}
	return nil
}

// Has reports whether any term states the given element.
func (t Terms) Has(element string) bool {
	for _, term := range t {
		if term.Element == element {
			return true
		}
	}
	return false
}

// GetLocalIdentifier returns the value of the dcterms:identifier term —
// the producer's local catalog/inventory number ("" when absent or the
// scheme is unknown). Mirrors Description.GetLocalIdentifier.
func (t Terms) GetLocalIdentifier(scheme string) string {
	if scheme != "dcterms" {
		return ""
	}
	for _, term := range t {
		if term.Element == "dcterms:identifier" {
			return term.Value
		}
	}
	return ""
}

// SetObjectIdentifier replaces the dcterms:identifier term's value in
// place (a no-op when the terms carry none). Unlike Description, Terms
// keeps no second identifier slot: read the local identifier with
// GetLocalIdentifier before swapping in the object identifier. What to do
// with each identifier is assembly policy, not data behavior.
func (t Terms) SetObjectIdentifier(id string) {
	for i, term := range t {
		if term.Element == "dcterms:identifier" {
			t[i].Value = id
			return
		}
	}
}
