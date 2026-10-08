// Package mapping holds one mapping per profile for the input folder. Each
// type implements input.Mapper and maps the terms of a description.csv
// onto the profile's description. It is the only package under cli/input
// that imports the profile packages, so input.Read takes a mapper as a
// value.
package mapping
