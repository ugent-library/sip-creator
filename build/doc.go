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
// [New] returns a [Builder] for one profile. [Builder.Build] validates a
// source package against the profile's rules and writes the package
// directory; a source package that fails validation or assembly writes
// nothing. Zipping the directory is a separate step, in the archive
// package.
//
// The Go type of a description belongs to its profile: eark.Terms,
// earkmods.Record or meemoo.Terms, or a [DescriptiveDocument] for a
// finished document where the profile accepts one. A profile for another
// descriptive standard implements [DescriptionEncoder] and exports its own
// [Definition].
package build
