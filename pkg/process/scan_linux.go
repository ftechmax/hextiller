//go:build linux

package process

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

type memRegion struct {
	start    uintptr
	end      uintptr
	readable bool
	writable bool
}

func parseMaps(pid uint32) ([]memRegion, error) {
	f, err := os.Open(fmt.Sprintf("/proc/%d/maps", pid))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var regions []memRegion
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for scanner.Scan() {
		region, ok := parseMapsLine(scanner.Text())
		if !ok {
			continue
		}
		regions = append(regions, region)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return regions, nil
}

// parseMapsLine parses a single /proc/[pid]/maps line.
// Format: "<start>-<end> <perms> <offset> <dev> <inode> [pathname]"
// Skip [vsyscall] since process_vm_readv returns EIO for it on x86_64.
func parseMapsLine(line string) (memRegion, bool) {
	fields := strings.Fields(line)
	if len(fields) < 5 {
		return memRegion{}, false
	}
	if len(fields) >= 6 && fields[5] == "[vsyscall]" {
		return memRegion{}, false
	}

	rng := fields[0]
	dash := strings.IndexByte(rng, '-')
	if dash < 0 {
		return memRegion{}, false
	}
	start, err := strconv.ParseUint(rng[:dash], 16, 64)
	if err != nil {
		return memRegion{}, false
	}
	end, err := strconv.ParseUint(rng[dash+1:], 16, 64)
	if err != nil {
		return memRegion{}, false
	}

	perms := fields[1]
	if len(perms) < 2 {
		return memRegion{}, false
	}
	return memRegion{
		start:    uintptr(start),
		end:      uintptr(end),
		readable: perms[0] == 'r',
		writable: perms[1] == 'w',
	}, true
}

func (p *Process) scanNumeric(size int, match func([]byte) bool, maxResults int, writableOnly bool) ([]uintptr, error) {
	if p == nil || p.PID == 0 {
		return nil, errors.New("process not initialized")
	}

	regions, err := parseMaps(p.PID)
	if err != nil {
		return nil, err
	}

	var (
		matches  []uintptr
		buf      []byte
		maxChunk = uintptr(1 << 20)
	)

	for _, r := range regions {
		if !r.readable {
			continue
		}
		if writableOnly && !r.writable {
			continue
		}
		offset := r.start
		for offset < r.end {
			chunk := r.end - offset
			if chunk > maxChunk {
				chunk = maxChunk
			}
			if cap(buf) < int(chunk) {
				buf = make([]byte, chunk)
			} else {
				buf = buf[:chunk]
			}

			local := []unix.Iovec{{Base: &buf[0]}}
			local[0].SetLen(len(buf))
			remote := []unix.RemoteIovec{{Base: offset, Len: len(buf)}}
			n, err := unix.ProcessVMReadv(int(p.PID), local, remote, 0)
			if err == nil && n > 0 {
				b := buf[:n]
				for i := 0; i+size <= len(b); i += size {
					if match(b[i : i+size]) {
						matches = append(matches, offset+uintptr(i))
						if maxResults > 0 && len(matches) >= maxResults {
							return matches, nil
						}
					}
				}
			}
			offset += chunk
		}
	}

	return matches, nil
}
