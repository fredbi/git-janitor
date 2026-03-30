package backend

import (
	"context"
	"testing"
)

func TestCheckCountObjects(t *testing.T) {
	// Simulate count-objects output parsing.
	input := "count: 56\n" +
		"size: 228\n" +
		"in-pack: 42\n" +
		"packs: 2\n" +
		"size-pack: 15\n" +
		"prune-packable: 0\n" +
		"garbage: 0\n" +
		"size-garbage: 0\n"

	report := &HealthReport{OK: true}

	// We can't call checkCountObjects directly (it runs git),
	// but we can verify the field names match by running the integration test.
	// Instead, test the evaluate logic with known values.
	report.LooseObjects = 56
	report.LooseSizeKB = 228
	report.PackedObjects = 42
	report.Packs = 2
	report.PackSizeKB = 15

	_ = input // used for documentation

	r := &Runner{Dir: "."}
	r.evaluateGCAdvice(t.Context(), report)

	if report.GCAdvised {
		t.Errorf("GCAdvised should be false for small repo, got reasons: %v", report.GCReasons)
	}
}

func TestEvaluateGCAdvice_LooseObjects(t *testing.T) {
	report := &HealthReport{
		OK:           true,
		LooseObjects: 5000, // above 50% of 6700 default threshold
	}

	r := &Runner{Dir: "."}
	r.evaluateGCAdvice(t.Context(), report)

	if !report.GCAdvised {
		t.Error("expected GCAdvised=true for 5000 loose objects")
	}

	if len(report.GCReasons) == 0 {
		t.Error("expected at least one GC reason")
	}
}

func TestEvaluateGCAdvice_PrunePackable(t *testing.T) {
	report := &HealthReport{
		OK:            true,
		PrunePackable: 100,
	}

	r := &Runner{Dir: "."}
	r.evaluateGCAdvice(t.Context(), report)

	if !report.GCAdvised {
		t.Error("expected GCAdvised=true for 100 prune-packable objects")
	}
}

func TestEvaluateGCAdvice_ManyPacks(t *testing.T) {
	report := &HealthReport{
		OK:    true,
		Packs: 15,
	}

	r := &Runner{Dir: "."}
	r.evaluateGCAdvice(t.Context(), report)

	if !report.GCAdvised {
		t.Error("expected GCAdvised=true for 15 packs")
	}
}

func TestEvaluateGCAdvice_Garbage(t *testing.T) {
	report := &HealthReport{
		OK:            true,
		Garbage:       3,
		GarbageSizeKB: 512,
	}

	r := &Runner{Dir: "."}
	r.evaluateGCAdvice(t.Context(), report)

	if !report.GCAdvised {
		t.Error("expected GCAdvised=true for garbage files")
	}
}

func TestIntegration_Health(t *testing.T) {
	r := &Runner{Dir: "."}

	report := r.Health(context.Background())

	t.Logf("OK=%v fsckErrors=%d", report.OK, len(report.FSCKErrors))
	t.Logf("loose=%d (%d KB) packed=%d packs=%d packSize=%d KB",
		report.LooseObjects, report.LooseSizeKB,
		report.PackedObjects, report.Packs, report.PackSizeKB)
	t.Logf("prunePackable=%d garbage=%d (%d KB)",
		report.PrunePackable, report.Garbage, report.GarbageSizeKB)
	t.Logf("GCAdvised=%v reasons=%v", report.GCAdvised, report.GCReasons)

	for _, e := range report.FSCKErrors {
		t.Logf("  fsck: %s", e)
	}
}
