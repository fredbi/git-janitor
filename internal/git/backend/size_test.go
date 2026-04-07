package backend

import (
	"context"
	"testing"

	"github.com/fredbi/git-janitor/internal/models"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1610612736, "1.5 GB"},
	}

	for _, tt := range tests {
		got := models.FormatBytes(tt.input)
		if got != tt.want {
			t.Errorf("models.FormatBytes(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIntegration_Size(t *testing.T) {
	r := &Runner{Dir: "."}

	s := r.Size(context.Background())

	t.Logf(".git size: %s (%d bytes)", models.FormatBytes(s.GitDirBytes), s.GitDirBytes)
	t.Logf("reachable: %s (%d bytes)", models.FormatBytes(s.ReachableBytes), s.ReachableBytes)
	t.Logf("repackAdvised=%v reasons=%v", s.RepackAdvised, s.RepackReasons)

	if s.GitDirBytes == 0 {
		t.Error("expected GitDirBytes > 0")
	}

	if s.ReachableBytes == 0 {
		t.Error("expected ReachableBytes > 0")
	}
}
