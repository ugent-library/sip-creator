package cli

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/ugent-library/sip-creator/profiles"
)

// runCLI runs the command line with args, as Run does but logging
// nowhere, and returns what it wrote to stdout and stderr and the error it
// ended with. Cobra keeps flag values on the commands, which are package
// variables, so every flag is set back to its default afterwards: a
// --no-zip in one run must not carry into the next.
func runCLI(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	var out, errOut bytes.Buffer
	logger = slog.New(slog.DiscardHandler)
	rootCmd.SetArgs(args)
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)
	t.Cleanup(func() {
		logger = nil
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
		for _, cmd := range rootCmd.Commands() {
			cmd.Flags().VisitAll(func(f *pflag.Flag) {
				if err := f.Value.Set(f.DefValue); err != nil {
					t.Errorf("reset --%s: %v", f.Name, err)
				}
				f.Changed = false
			})
		}
	})
	err = rootCmd.Execute()
	return out.String(), errOut.String(), err
}

// writeFolder builds an input folder from slash paths and contents.
func writeFolder(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for p, content := range files {
		abs := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// A folder that breaks no rule is summarized on stdout, and check needs no
// configuration to say so (ADR-0010): the submitter settings create
// requires are empty here. The summary's wording and counts are not
// pinned: what the summary counts is still to be decided.
func TestCheckSummarizesAValidFolder(t *testing.T) {
	t.Setenv("SIP_SUBMITTER_NAME", "")
	t.Setenv("SIP_SUBMITTER_OR_ID", "")
	root := writeFolder(t, map[string]string{
		"description.csv":                   "key,value\nidentifier,ID-1\ntitle,Test\n",
		"documentation/manual.pdf":          "m",
		"representations/master/a.tif":      "a",
		"representations/master/sub/b.tif":  "b",
		"representations/access/access.pdf": "c",
	})

	stdout, stderr, err := runCLI(t, "check", "--profile", "eark", root)
	if err != nil {
		t.Fatalf("check: %v\n%s", err, stderr)
	}
	if !strings.HasPrefix(stdout, "OK: ") {
		t.Errorf("stdout = %q, want a summary starting with %q", stdout, "OK: ")
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want nothing", stderr)
	}
}

// The example folders are what producers copy, so check accepts each one
// under its own profile.
func TestCheckAcceptsTheExamples(t *testing.T) {
	for _, name := range profiles.Names() {
		_, stderr, err := runCLI(t, "check", "--profile", name, filepath.Join("..", "examples", name))
		if err != nil {
			t.Errorf("check --profile %s on its example: %v\n%s", name, err, stderr)
		}
	}
}

// Every violation of the input specification is printed on its own line
// to stderr, and the command ends with one summary naming the folder and
// the count. Nothing reaches stdout.
func TestCheckReportsEveryViolation(t *testing.T) {
	root := writeFolder(t, map[string]string{
		// no description.csv
		"stray.tif":                    "x", // content beside representations/
		"representations/master/a.tif": "a",
	})

	stdout, stderr, err := runCLI(t, "check", "--profile", "eark", root)
	if want := root + ": 2 problem(s) found"; err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
	lines := strings.Split(strings.TrimSuffix(stderr, "\n"), "\n")
	if len(lines) != 2 ||
		!strings.Contains(stderr, "stray.tif: content must live inside representations/") ||
		!strings.Contains(stderr, "descriptive metadata is missing") {
		t.Errorf("stderr = %q, want one line per violation", stderr)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want nothing", stdout)
	}
}

// After the input specification, check applies the profile's rules on the
// package, as a build would: two representations pass the folder rules
// but not Meemoo's basic profile, which allows one.
func TestCheckAppliesTheProfileRules(t *testing.T) {
	root := writeFolder(t, map[string]string{
		"description.csv":              "key,value\nidentifier,ID-1\ntitle,Test\ndescription,Beschrijving\ncreated,2026\n",
		"representations/master/a.tif": "a",
		"representations/access/b.pdf": "b",
	})

	stdout, _, err := runCLI(t, "check", "--profile", "basic", root)
	if want := `profile "basic" allows at most 1 representation(s), the package has 2`; err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("error = %v, want %q", err, want)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want nothing", stdout)
	}
}

// What stops check before any rule runs is returned as it is: an unknown
// profile names the ones there are, and a missing folder is named.
func TestCheckRefusesWhatItCannotRead(t *testing.T) {
	_, _, err := runCLI(t, "check", "--profile", "nope", t.TempDir())
	if want := `unknown profile "nope" (available: ` + strings.Join(profiles.Names(), ", ") + ")"; err == nil || err.Error() != want {
		t.Errorf("error = %v, want %q", err, want)
	}

	missing := filepath.Join(t.TempDir(), "missing")
	_, _, err = runCLI(t, "check", "--profile", "eark", missing)
	if err == nil || !strings.Contains(err.Error(), "input folder: ") || !strings.Contains(err.Error(), "no such file") {
		t.Errorf("error = %v, want the missing input folder named", err)
	}
}
