package auditor

import (
	"runtime"
	"testing"

	"github.com/alexandrmotologa/reconcileai/internal/config"
	"github.com/alexandrmotologa/reconcileai/internal/synth"
)

func BenchmarkReconcileDossier(b *testing.B) {
	dossier, err := synth.BuildQtyOverbillingDossier(b.TempDir())
	if err != nil {
		b.Fatalf("failed to build test dossier: %v", err)
	}

	cfg := config.DefaultConfig()

	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		summary := ReconcileDossier(dossier, cfg)
		if summary == nil {
			b.Fatal("nil summary")
		}
	}
	b.StopTimer()

	runtime.ReadMemStats(&m2)
	allocMB := float64(m2.Alloc) / 1024 / 1024
	b.Logf("Final heap alloc: %.2f MB (Target: < 25 MB)", allocMB)
}
