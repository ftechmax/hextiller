//go:build windows

package process

import "golang.org/x/sys/windows"

type Process struct {
	Handle windows.Handle
	PID    uint32
}

func Open(pid uint32) (*Process, error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_INFORMATION|windows.PROCESS_VM_READ|windows.PROCESS_VM_WRITE|windows.PROCESS_VM_OPERATION, false, pid)
	if err != nil {
		return nil, err
	}
	return &Process{Handle: h, PID: pid}, nil
}

func (p *Process) Close() error {
	if p == nil || p.Handle == 0 {
		return nil
	}
	return windows.CloseHandle(p.Handle)
}
