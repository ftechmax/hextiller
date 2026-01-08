//go:build windows

package process

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

type Info struct {
	PID       uint32
	ParentPID uint32
	Exe       string
}

func List() ([]Info, error) {
	snapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snapshot)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))

	if err := windows.Process32First(snapshot, &entry); err != nil {
		return nil, err
	}

	processes := make([]Info, 0, 128)
	for {
		exe := windows.UTF16ToString(entry.ExeFile[:])
		processes = append(processes, Info{
			PID:       entry.ProcessID,
			ParentPID: entry.ParentProcessID,
			Exe:       exe,
		})

		if err := windows.Process32Next(snapshot, &entry); err != nil {
			if err == windows.ERROR_NO_MORE_FILES {
				break
			}
			return nil, err
		}
	}

	return processes, nil
}
