// Package build turns a source package into an E-ARK Submission Information
// Package (SIP) on disk: the content files and their descriptive metadata,
// together with the METS and PREMIS documents and the schemas they point at.
//
// A build takes three values. A [Definition] is the profile: the
// specification the package follows and the standard its descriptive
// metadata is written in. The profiles in this module come from
// profiles.Get, and [Definition.WithSubmitter] adds the submitting
// organization. A [Config] names the profile, the directory packages are
// written to and the logger. A [SourcePackage] holds what one package is
// built from: its description, its representations and their files.
//
// [New] returns a [Builder] for one profile, and [Builder.Build] builds one
// package from a source package. A source package is the package as you
// supply it: paths to your files, your names for the representations, and
// the description. Build validates it against the profile's rules, then
// turns it into the package graph of package sip: the same package as it
// will be written, with its identifiers minted, every file's path inside
// the package, and the METS and PREMIS documents it carries. It then writes
// the graph to disk, filling in each file's size and checksum as it is
// written, and returns it. A source package that fails validation or
// assembly writes nothing. Zipping the directory is a separate step, in the
// archive package.
//
// The Go type of a description belongs to the profile's model: ugent.Terms,
// ugent.Record or meemoo.Terms, or an [EncodedDescription] for a
// finished document where the profile accepts one. A profile for another
// descriptive standard implements [MetadataModel] and exports its own
// [Definition].
package build
