// Package mapping holds the profiles' mappings for the input folder: one
// type per profile implementing input.Mapper, mapping the terms of a
// description.csv onto the profile's description. It is the one place in
// cli/input that imports the profile packages for descriptive metadata;
// Read imports none and takes a mapper as a value, and the CLI decides
// which mapping a profile uses.
package mapping
