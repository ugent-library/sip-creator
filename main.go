// Sip-creator builds E-ARK Submission Information Packages (SIPs) from an
// input folder: content files plus their descriptive metadata.
//
// Usage:
//
//	sip-creator create --profile <profile> <input> <destination>
//	sip-creator check --profile <profile> <input>
//
// The profiles are eark, eark-mods and basic. The README describes the
// profiles, the input folder and the environment variables create needs.
package main

import "github.com/ugent-library/sip-creator/cli"

func main() {
	cli.Run()
}
