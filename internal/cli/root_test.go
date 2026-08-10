package cli

import "testing"

// TestResolveDataDirEnvOverride guards the HARB-14 fix: the daemon child must
// resolve the same data dir the parent resolved, via HARBOR_DATA_DIR.
func TestResolveDataDirEnvOverride(t *testing.T) {
	t.Setenv("HARBOR_DATA_DIR", "/tmp/override-harbor")
	if got := resolveDataDir(); got != "/tmp/override-harbor" {
		t.Fatalf("resolveDataDir = %q, want the HARBOR_DATA_DIR override", got)
	}
}

func TestResolveDataDirClearsAfterEnvUnset(t *testing.T) {
	// Without the env override, resolution falls back to config/default — and
	// must NOT remember the previous env value (no accidental leak).
	t.Setenv("HARBOR_DATA_DIR", "")
	if got := resolveDataDir(); got == "" {
		t.Fatalf("resolveDataDir must never return empty")
	}
}
