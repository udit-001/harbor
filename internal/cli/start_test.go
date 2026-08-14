package cli

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// TestWaitForServerSucceedsWhenPortAccepts guards the sleep→poll fix: the
// background daemon binds in ~15ms, so waiting on a live server must return
// near-instantly — not after a hardcoded 500ms sleep.
func TestWaitForServerSucceedsWhenPortAccepts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	port := srv.Listener.Addr().(*net.TCPAddr).Port

	start := time.Now()
	if err := waitForServer(port, 2*time.Second); err != nil {
		t.Fatalf("waitForServer on a live server: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("waitForServer took %s on a live server; want near-instant", elapsed)
	}
}

// TestWaitForServerTimesOutWhenNothingListens guards the failure path: with
// no listener on the port, waitForServer must give up after the timeout
// instead of polling forever.
func TestWaitForServerTimesOutWhenNothingListens(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()

	start := time.Now()
	err = waitForServer(port, 200*time.Millisecond)
	if err == nil {
		t.Fatal("waitForServer on a closed port must time out with an error")
	}
	if elapsed := time.Since(start); elapsed < 200*time.Millisecond {
		t.Fatalf("waitForServer returned after %s; must respect the timeout", elapsed)
	}
}
