package profiles

import "testing"

func TestWithSubmitterMeemoo(t *testing.T) {
	def := basicDef(t)

	got, err := def.WithSubmitter("Universiteitsbibliotheek Gent", "OR-a1b2c3d")
	if err != nil {
		t.Fatalf("WithSubmitter() error = %v", err)
	}

	agents := got.Mets.Agents
	if len(agents) != len(def.Mets.Agents)+1 {
		t.Fatalf("agents = %d, want %d", len(agents), len(def.Mets.Agents)+1)
	}
	sub := agents[len(agents)-1]
	if sub.Role != "CREATOR" || sub.Type != "ORGANIZATION" {
		t.Errorf("submitter agent role/type = %q/%q, want CREATOR/ORGANIZATION", sub.Role, sub.Type)
	}
	if sub.Name != "Universiteitsbibliotheek Gent" {
		t.Errorf("submitter name = %q", sub.Name)
	}
	if sub.Note != "OR-a1b2c3d" || sub.NoteType != "IDENTIFICATIONCODE" {
		t.Errorf("submitter note = %q (%q), want the OR-id as IDENTIFICATIONCODE", sub.Note, sub.NoteType)
	}
}

func TestWithSubmitterMeemooRequiresORID(t *testing.T) {
	if _, err := basicDef(t).WithSubmitter("Universiteitsbibliotheek Gent", ""); err == nil {
		t.Fatal("WithSubmitter() with empty OR-id on a meemoo profile: want error, got nil")
	}
}

func TestWithSubmitterRequiresName(t *testing.T) {
	if _, err := basicDef(t).WithSubmitter("", "OR-a1b2c3d"); err == nil {
		t.Fatal("WithSubmitter() with empty name: want error, got nil")
	}
}

func TestWithSubmitterEARK(t *testing.T) {
	def, ok := Get("eark")
	if !ok {
		t.Fatal(`no "eark" definition registered`)
	}

	// The OR-id is a meemoo concept; a configured value is ignored here.
	got, err := def.WithSubmitter("Universiteitsbibliotheek Gent", "OR-a1b2c3d")
	if err != nil {
		t.Fatalf("WithSubmitter() error = %v", err)
	}

	sub := got.Mets.Agents[len(got.Mets.Agents)-1]
	if sub.Name != "Universiteitsbibliotheek Gent" {
		t.Errorf("submitter name = %q", sub.Name)
	}
	if sub.Note != "" || sub.NoteType != "" {
		t.Errorf("eark submitter note = %q (%q), want none", sub.Note, sub.NoteType)
	}
}

func TestWithSubmitterLeavesRegistryUntouched(t *testing.T) {
	before := len(basicDef(t).Mets.Agents)

	if _, err := basicDef(t).WithSubmitter("Universiteitsbibliotheek Gent", "OR-a1b2c3d"); err != nil {
		t.Fatalf("WithSubmitter() error = %v", err)
	}

	after := basicDef(t)
	if len(after.Mets.Agents) != before {
		t.Fatalf("registry agents = %d after WithSubmitter, want %d", len(after.Mets.Agents), before)
	}
	for _, a := range after.Mets.Agents {
		if a.Type == "ORGANIZATION" {
			t.Errorf("registry gained an ORGANIZATION agent: %+v", a)
		}
	}
}
