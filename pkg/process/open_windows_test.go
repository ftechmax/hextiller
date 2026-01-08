//go:build windows

package process

import "testing"

func TestOpenCloseSelf(t *testing.T) {
	p := openSelf(t)
	if p.Handle == 0 {
		t.Fatalf("expected valid handle")
	}
}
