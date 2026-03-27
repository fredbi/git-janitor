package git

import (
	"testing"
)

func TestParseStatus_BranchHeaders(t *testing.T) {
	input := `# branch.oid abc123def456
# branch.head main
# branch.upstream origin/main
# branch.ab +3 -1
`
	s := parseStatus(input)

	if s.OID != "abc123def456" {
		t.Errorf("OID = %q, want %q", s.OID, "abc123def456")
	}

	if s.Branch != "main" {
		t.Errorf("Branch = %q, want %q", s.Branch, "main")
	}

	if s.Upstream != "origin/main" {
		t.Errorf("Upstream = %q, want %q", s.Upstream, "origin/main")
	}

	if s.Ahead != 3 {
		t.Errorf("Ahead = %d, want %d", s.Ahead, 3)
	}

	if s.Behind != 1 {
		t.Errorf("Behind = %d, want %d", s.Behind, 1)
	}
}

func TestParseStatus_DetachedHead(t *testing.T) {
	input := `# branch.oid abc123
# branch.head (detached)
`
	s := parseStatus(input)

	if s.Branch != "" {
		t.Errorf("Branch = %q, want empty for detached HEAD", s.Branch)
	}
}

func TestParseStatus_OrdinaryEntries(t *testing.T) {
	input := `# branch.oid abc123
# branch.head main
1 .M N... 100644 100644 100644 abc123 def456 src/main.go
1 A. N... 000000 100644 100644 000000 abc123 new_file.go
`
	s := parseStatus(input)

	if len(s.Entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(s.Entries))
	}

	if s.Entries[0].XY != ".M" || s.Entries[0].Path != "src/main.go" {
		t.Errorf("entry[0] = {%q, %q}, want {.M, src/main.go}", s.Entries[0].XY, s.Entries[0].Path)
	}

	if s.Entries[1].XY != "A." || s.Entries[1].Path != "new_file.go" {
		t.Errorf("entry[1] = {%q, %q}, want {A., new_file.go}", s.Entries[1].XY, s.Entries[1].Path)
	}
}

func TestParseStatus_UntrackedAndIgnored(t *testing.T) {
	input := `# branch.oid abc123
# branch.head main
? untracked.txt
! ignored.log
`
	s := parseStatus(input)

	if len(s.Entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(s.Entries))
	}

	if !s.Entries[0].IsUntracked() {
		t.Error("entry[0] should be untracked")
	}

	if s.Entries[0].Path != "untracked.txt" {
		t.Errorf("entry[0].Path = %q, want %q", s.Entries[0].Path, "untracked.txt")
	}

	if !s.Entries[1].IsIgnored() {
		t.Error("entry[1] should be ignored")
	}
}

func TestParseStatus_RenameEntry(t *testing.T) {
	input := `# branch.oid abc123
# branch.head main
2 R. N... 100644 100644 100644 abc123 def456 R100 new_name.go	old_name.go
`
	s := parseStatus(input)

	if len(s.Entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(s.Entries))
	}

	e := s.Entries[0]
	if e.XY != "R." {
		t.Errorf("XY = %q, want %q", e.XY, "R.")
	}

	if e.Path != "new_name.go" {
		t.Errorf("Path = %q, want %q", e.Path, "new_name.go")
	}

	if e.OrigPath != "old_name.go" {
		t.Errorf("OrigPath = %q, want %q", e.OrigPath, "old_name.go")
	}
}

func TestParseStatus_Empty(t *testing.T) {
	s := parseStatus("")

	if s.IsDirty() {
		t.Error("empty status should not be dirty")
	}
}

func TestStatus_IsDirty(t *testing.T) {
	clean := Status{}
	if clean.IsDirty() {
		t.Error("Status with no entries should not be dirty")
	}

	dirty := Status{Entries: []StatusEntry{{XY: ".M", Path: "foo.go"}}}
	if !dirty.IsDirty() {
		t.Error("Status with entries should be dirty")
	}
}
