//go:build linux

package process

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Info struct {
	PID       uint32
	ParentPID uint32
	Exe       string
}

func List() ([]Info, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	processes := make([]Info, 0, 128)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid64, err := strconv.ParseUint(e.Name(), 10, 32)
		if err != nil {
			continue
		}
		pid := uint32(pid64)
		info := Info{PID: pid}

		if target, err := os.Readlink(fmt.Sprintf("/proc/%d/exe", pid)); err == nil && target != "" {
			info.Exe = filepath.Base(target)
			if isWineLoader(info.Exe) {
				if name := resolveWineExe(pid); name != "" {
					info.Exe = name
				}
			}
		} else if comm, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", pid)); err == nil {
			info.Exe = strings.TrimSpace(string(comm))
		}

		if stat, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid)); err == nil {
			if ppid, ok := parsePPIDFromStat(stat); ok {
				info.ParentPID = ppid
			}
		}

		processes = append(processes, info)
	}

	return processes, nil
}

// isWineLoader reports whether exe is the basename of a Wine/Proton loader
// that fronts a Windows .exe (the real target is in /proc/[pid]/cmdline).
func isWineLoader(exe string) bool {
	switch exe {
	case "wine", "wine64", "wine-preloader", "wine64-preloader":
		return true
	}
	return false
}

// resolveWineExe reads /proc/[pid]/cmdline and returns the basename of the
// first argument ending in .exe. Returns "" if none is found.
func resolveWineExe(pid uint32) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return ""
	}
	return resolveWineExeFromCmdline(data)
}

// resolveWineExeFromCmdline parses NUL-separated cmdline bytes and returns the
// basename of the first argument ending in .exe. Windows-style paths may use
// backslashes (e.g. "Z:\\home\\user\\Game.exe"), so trim on '\\' first.
func resolveWineExeFromCmdline(data []byte) string {
	for _, arg := range bytes.Split(data, []byte{0}) {
		if len(arg) < 4 {
			continue
		}
		s := string(arg)
		if !strings.EqualFold(s[len(s)-4:], ".exe") {
			continue
		}
		if i := strings.LastIndexByte(s, '\\'); i >= 0 {
			s = s[i+1:]
		}
		return filepath.Base(s)
	}
	return ""
}

// parsePPIDFromStat returns field 4 (ppid) from /proc/[pid]/stat.
// Format: "<pid> (<comm>) <state> <ppid> ..." where <comm> may contain spaces and parens,
// so scan fields only after the final ')'.
func parsePPIDFromStat(stat []byte) (uint32, bool) {
	end := bytes.LastIndexByte(stat, ')')
	if end < 0 || end+1 >= len(stat) {
		return 0, false
	}
	fields := strings.Fields(string(stat[end+1:]))
	if len(fields) < 2 {
		return 0, false
	}
	ppid, err := strconv.ParseUint(fields[1], 10, 32)
	if err != nil {
		return 0, false
	}
	return uint32(ppid), true
}
