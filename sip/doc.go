// Package sip is the domain model of a Submission Information Package
// (SIP): the package, the intellectual entity it describes, its
// representations and files, the agents and other METS values a profile
// declares, and the identifier scheme (uuid-<uuid>). The graph types
// (Package, Entity, Representation, File) are plain data with no checks of
// their own; what a caller supplies is validated in package build before a
// graph is assembled.
package sip
