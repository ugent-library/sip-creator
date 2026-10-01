// Package sip is the domain model of a Submission Information Package
// (SIP): the package, the intellectual entity it describes, its
// representations and files, the agents and other METS values a profile
// declares, and the identifier scheme (uuid-<uuid>). The graph types
// (Package, Entity, Representation, File) describe a package as it is
// written to disk. Package build assembles them from a build.SourcePackage,
// which it validates first, so the graph types carry no checks of their
// own.
package sip
