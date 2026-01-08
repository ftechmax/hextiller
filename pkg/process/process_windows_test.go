//go:build windows

package process

import (
	"os"
	"testing"
)

func TestListReturnsProcesses(t *testing.T) {
	procs, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(procs) == 0 {
		t.Fatalf("expected at least one process")
	}
	selfPID := uint32(os.Getpid())
	foundSelf := false
	for _, p := range procs {
		if p.PID == selfPID {
			foundSelf = true
			break
		}
	}
	if !foundSelf {
		t.Fatalf("current pid %d not found in process list", selfPID)
	}
}
