package render

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// The comment surface's decision core is plain JS (pageview-core.js) with a
// node:test suite beside it. Go can't execute it, so this wrapper shells out
// to `node --test` when node is available and skips otherwise — CI stays
// green on machines without node, and the suite runs everywhere it can.
func TestPageviewCoreJS(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available; skipping pageview-core JS tests")
	}
	if _, err := os.Stat("pageview-core.test.js"); err != nil {
		t.Skip("test file not present in working dir")
	}
	out, err := exec.Command(node, "--test", "pageview-core.test.js").CombinedOutput()
	if err != nil {
		t.Fatalf("node --test failed:\n%s", out)
	}
	if !contains(out, "# fail 0") {
		t.Fatalf("JS suite did not report zero failures:\n%s", out)
	}
}

func contains(b []byte, sub string) bool {
	return strings.Contains(string(b), sub)
}
