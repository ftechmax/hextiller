//go:build linux

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

func TestIsWineLoader(t *testing.T) {
	cases := map[string]bool{
		"wine":             true,
		"wine64":           true,
		"wine-preloader":   true,
		"wine64-preloader": true,
		"wineserver":       false,
		"Game.exe":         false,
		"bash":             false,
		"":                 false,
	}
	for in, want := range cases {
		if got := isWineLoader(in); got != want {
			t.Errorf("isWineLoader(%q)=%v want %v", in, got, want)
		}
	}
}

func TestResolveWineExeFromCmdline(t *testing.T) {
	nul := "\x00"
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			"proton windows path",
			"wine64-preloader" + nul + `Z:\home\user\Games\MyGame.exe` + nul + "--arg" + nul,
			"MyGame.exe",
		},
		{
			"pure windows path",
			"wine" + nul + `C:\Program Files\Foo\Bar.EXE` + nul,
			"Bar.EXE",
		},
		{
			"unix path fallback",
			"wine" + nul + "/home/user/game.exe" + nul,
			"game.exe",
		},
		{
			"bare exe",
			"wine" + nul + "Game.exe" + nul,
			"Game.exe",
		},
		{
			"launcher picked first",
			"wine64-preloader" + nul + `C:\Launcher.exe` + nul + "--launch" + nul + "Game.exe" + nul,
			"Launcher.exe",
		},
		{"no exe arg", "wine" + nul + "--help" + nul, ""},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := resolveWineExeFromCmdline([]byte(tc.in)); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestParsePPIDFromStat(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want uint32
		ok   bool
	}{
		{"simple", "1234 (cat) R 1000 1234 1234 34816 1234 4194304", 1000, true},
		{"comm with spaces and parens", "1234 (foo ) bar) S 42 1234 1234", 42, true},
		{"truncated", "1234 (cat)", 0, false},
		{"missing paren", "1234 cat R 1000", 0, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parsePPIDFromStat([]byte(tc.in))
			if ok != tc.ok {
				t.Fatalf("ok=%v want %v", ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Fatalf("ppid=%d want %d", got, tc.want)
			}
		})
	}
}
