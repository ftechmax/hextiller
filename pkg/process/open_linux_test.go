//go:build linux

package process

import (
	"os"
	"testing"
)

func TestOpenCloseSelf(t *testing.T) {
	p := openSelf(t)
	if p.PID != uint32(os.Getpid()) {
		t.Fatalf("expected PID %d, got %d", os.Getpid(), p.PID)
	}
}
