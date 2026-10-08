// Package sip is the domain model of a Submission Information Package
// (SIP): the package, the intellectual entity it describes, its
// representations and files, the agents and other METS values a profile
// declares, and the identifier scheme (uuid-<uuid>). The graph types
// (Package, Entity, Representation, File) describe a package as it is
// written to disk. In an assembled package, every node has an identifier,
// a path and a media type. The graph types carry no checks of their own.
// Package build assembles them and keeps these conditions.
package sip
