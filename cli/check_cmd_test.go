package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/ugent-library/sip-creator/profiles"
)

// runCLI runs the command line with args, as Run does, and returns what it wrote to stdout and stderr and the error it
// ended with.
func runCLI(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	_, stdout, stderr, err = runCommand(t, args...)
	return stdout, stderr, err
}

// runCommand is runCLI that also returns the command that ran, which Run
// needs for the exit status. Cobra keeps flag values on the commands,
// which are package variables, so every flag is set back to its default
// afterwards: a --no-zip in one run must not carry into the next.
func runCommand(t *testing.T, args ...string) (cmd *cobra.Command, stdout, stderr string, err error) {
	t.Helper()
	var out, errOut bytes.Buffer
	rootCmd.SetArgs(args)
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)
	t.Cleanup(func() {
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
	cmd, err = rootCmd.ExecuteC()
	return cmd, out.String(), errOut.String(), err
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

// A folder that breaks no rule gets the OK verdict on stdout, and check
// needs no configuration to say so (ADR-0010): the submitter settings
// create requires are empty here.
func TestCheckSummarizesAValidFolder(t *testing.T) {
	t.Setenv("SIP_SUBMITTER_NAME", "")
	t.Setenv("SIP_SUBMITTER_OR_ID", "")
	root := writeFolder(t, map[string]string{
		"description.csv":                    "key,value\nidentifier,ID-1\ntitle,Test\n",
		"documentation/manual.pdf":           "m",
		"representations/archival/a.tif":     "a",
		"representations/archival/sub/b.tif": "b",
		"representations/access/access.pdf":  "c",
	})

	stdout, stderr, err := runCLI(t, "check", "--profile", "ugent/basic", root)
	if err != nil {
		t.Fatalf("check: %v\n%s", err, stderr)
	}
	if want := "OK: the folder meets the input specification for profile ugent/basic.\n"; !strings.HasSuffix(stdout, want) {
		t.Errorf("stdout = %q, want it to end in %q", stdout, want)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want nothing", stderr)
	}
}

// The example folders are what producers copy, so check accepts each one
// under its own profile.
func TestCheckAcceptsTheExamples(t *testing.T) {
	for _, name := range profiles.Names() {
		_, stderr, err := runCLI(t, "check", "--profile", name, filepath.Join("..", "examples", filepath.FromSlash(name)))
		if err != nil {
			t.Errorf("check --profile %s on its example: %v\n%s", name, err, stderr)
		}
	}
}

// Every violation of the input specification is listed in the report on
// stdout, under a count, and the command ends with an error that names the
// folder and the count only. Nothing reaches stderr: Run prints the error.
func TestCheckReportsEveryViolation(t *testing.T) {
	root := writeFolder(t, map[string]string{
		// no description.csv
		"stray.tif":                    "x", // content beside representations/
		"representations/master/a.tif": "a",
	})

	stdout, stderr, err := runCLI(t, "check", "--profile", "ugent/basic", root)
	if want := root + ": 2 problem(s) found"; err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
	// The problems come first, then the summary of what could be read, then
	// the verdict.
	want := "2 problems in " + root + "\n\n" +
		"  descriptive metadata is missing: every package folder needs a description.csv or a dc.xml describing the content (input specification §3)\n" +
		"  stray.tif: content must live in a representation folder, representations/<name>/ (only the reserved names of the input specification may sit beside representations/)\n" +
		"\n" +
		"Input folder:         " + root + "\n" +
		"Profile:              ugent/basic\n" +
		"\n" +
		"Descriptive metadata: none\n" +
		"Representations:      1\n" +
		"Essence files:        1\n" +
		"Documentation files:  0\n" +
		"PREMIS files:         0\n" +
		"Format report:        not supplied (files carry no format information)\n" +
		"\n" +
		"FAILED: fix the problems listed at the top and run check again.\n"
	if stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want nothing", stderr)
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

	stdout, _, err := runCLI(t, "check", "--profile", "meemoo/basic", root)
	if want := root + ": 1 problem(s) found"; err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
	if want := `  profile "meemoo/basic" allows at most 1 representation(s), the package has 2`; !strings.Contains(stdout, want) {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
}

// The profile's rules run only on a folder read without violations: a
// folder with a broken description.csv and two representations under
// basic reports the description, not also the representation count.
func TestCheckSkipsTheProfileRulesOnAPartlyReadFolder(t *testing.T) {
	root := writeFolder(t, map[string]string{
		"description.csv":              "key,value\nnot-a-key,x\n",
		"representations/master/a.tif": "a",
		"representations/access/b.pdf": "b",
	})

	stdout, _, err := runCLI(t, "check", "--profile", "meemoo/basic", root)
	if err == nil {
		t.Fatal("check passed a folder with a broken description.csv")
	}
	if strings.Contains(stdout, "allows at most 1 representation") {
		t.Errorf("stdout = %q, want no profile rule findings", stdout)
	}
}

// Check reads received preservation files: one that is not well-formed XML
// fails check, not only create.
func TestCheckReportsMalformedPremis(t *testing.T) {
	root := writeFolder(t, map[string]string{
		"description.csv":              "key,value\nidentifier,ID-1\ntitle,Test\n",
		"representations/master/a.tif": "a",
		"premis/vendor.xml":            "<premis:premis>",
	})

	stdout, _, err := runCLI(t, "check", "--profile", "ugent/basic", root)
	if want := root + ": 1 problem(s) found"; err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
	if !strings.Contains(stdout, "premis/vendor.xml: not well-formed XML") {
		t.Errorf("stdout = %q, want the malformed file named", stdout)
	}
}

// A siegfried.json without an entry for a content file fails check, as it
// fails create, although check computes no checksum.
func TestCheckReportsContentMissingFromTheReport(t *testing.T) {
	root := writeFolder(t, map[string]string{
		"description.csv":              "key,value\nidentifier,ID-1\ntitle,Test\n",
		"representations/master/a.tif": "a",
		"representations/master/b.tif": "b",
		"siegfried.json":               `{"siegfried":"1.11.0","files":[{"filename":"representations/master/a.tif","md5":"0","matches":[]}]}`,
	})

	stdout, _, err := runCLI(t, "check", "--profile", "ugent/basic", root)
	if want := root + ": 1 problem(s) found"; err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
	if !strings.Contains(stdout, "siegfried.json has no entry for representations/master/b.tif") {
		t.Errorf("stdout = %q, want the file without an entry named", stdout)
	}
}

// The summary counts what the folder holds, across the package and its
// representations. The ugent/basic example describes its representation too.
func TestCheckSummarizesTheExample(t *testing.T) {
	stdout, _, err := runCLI(t, "check", "--profile", "ugent/basic", filepath.Join("..", "examples", "ugent", "basic"))
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	want := "Input folder:         ../examples/ugent/basic\n" +
		"Profile:              ugent/basic\n" +
		"\n" +
		"Descriptive metadata: description.csv\n" +
		"Representations:      2 (1 with its own description)\n" +
		"Essence files:        2\n" +
		"Documentation files:  2\n" +
		"PREMIS files:         2\n" +
		"Format report:        not supplied (files carry no format information)\n" +
		"\n" +
		"OK: the folder meets the input specification for profile ugent/basic.\n"
	if stdout != want {
		t.Errorf("stdout = %q, want %q", stdout, want)
	}
}

// A supplied document is named as the package description, and a supplied
// characterization report as the format report.
func TestCheckNamesSuppliedDocumentAndReport(t *testing.T) {
	root := writeFolder(t, map[string]string{
		"dc.xml":                         "<simpledc><identifier>ID-1</identifier><title>Test</title></simpledc>",
		"representations/archival/a.tif": "a",
		"siegfried.json":                 `{"siegfried":"1.11.0","files":[{"filename":"representations/archival/a.tif","md5":"0","matches":[]}]}`,
	})

	stdout, _, err := runCLI(t, "check", "--profile", "ugent/basic", root)
	if err != nil {
		t.Fatalf("check: %v\n%s", err, stdout)
	}
	for _, want := range []string{
		"Descriptive metadata: dc.xml (supplied document, copied as it is)\n",
		"Format report:        siegfried.json\n",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout = %q, want it to contain %q", stdout, want)
		}
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
	_, _, err = runCLI(t, "check", "--profile", "ugent/basic", missing)
	if err == nil || !strings.Contains(err.Error(), "input folder: ") || !strings.Contains(err.Error(), "no such file") {
		t.Errorf("error = %v, want the missing input folder named", err)
	}
}

// check exits with 1 when the folder has problems and with 2 when it could
// not check the folder at all; create exits with 1 on any error.
func TestExitStatus(t *testing.T) {
	broken := writeFolder(t, map[string]string{"representations/master/a.tif": "a"}) // no description
	missing := filepath.Join(t.TempDir(), "missing")
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"check, folder with problems", []string{"check", "--profile", "ugent/basic", broken}, 1},
		{"check, missing folder", []string{"check", "--profile", "ugent/basic", missing}, 2},
		{"check, unknown profile", []string{"check", "--profile", "nope", broken}, 2},
		{"check, no folder given", []string{"check", "--profile", "ugent/basic"}, 2},
		{"create, missing folder", []string{"create", "--profile", "ugent/basic", missing, t.TempDir()}, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("SIP_SUBMITTER_NAME", "Test")
			t.Setenv("SIP_SUBMITTER_OR_ID", "OR-test")
			cmd, _, _, err := runCommand(t, tc.args...)
			if err == nil {
				t.Fatal("want an error")
			}
			if got := exitStatus(cmd, err); got != tc.want {
				t.Errorf("exit status = %d, want %d (error: %v)", got, tc.want, err)
			}
		})
	}
}
