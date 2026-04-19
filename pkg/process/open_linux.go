//go:build linux

package process

import (
	"fmt"
	"os"
)

type Process struct {
	PID uint32
}

func Open(pid uint32) (*Process, error) {
	if _, err := os.Stat(fmt.Sprintf("/proc/%d", pid)); err != nil {
		return nil, err
	}
	return &Process{PID: pid}, nil
}

func (p *Process) Close() error {
	return nil
}
