//go:build windows

package process

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
)

func (p *Process) scanNumeric(size int, match func([]byte) bool, maxResults int, writableOnly bool) ([]uintptr, error) {
	if p == nil || p.Handle == 0 {
		return nil, errors.New("process handle is nil")
	}

	var (
		matches  []uintptr
		addr     uintptr
		mbi      windows.MemoryBasicInformation
		buf      []byte
		maxChunk = uintptr(1 << 20)
	)

	for {
		if err := windows.VirtualQueryEx(p.Handle, addr, &mbi, unsafe.Sizeof(mbi)); err != nil {
			break
		}

		regionSize := uintptr(mbi.RegionSize)
		base := uintptr(mbi.BaseAddress)
		if regionSize == 0 {
			break
		}

		if mbi.State == windows.MEM_COMMIT && isReadable(mbi.Protect) && (mbi.Protect&windows.PAGE_GUARD) == 0 {
			if writableOnly && !isWritable(mbi.Protect) {
				addr = base + regionSize
				continue
			}
			end := base + regionSize
			offset := base
			for offset < end {
				chunk := end - offset
				if chunk > maxChunk {
					chunk = maxChunk
				}
				if cap(buf) < int(chunk) {
					buf = make([]byte, chunk)
				} else {
					buf = buf[:chunk]
				}

				var read uintptr
				if err := windows.ReadProcessMemory(p.Handle, offset, &buf[0], uintptr(len(buf)), &read); err == nil && read > 0 {
					b := buf[:read]
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

		addr = base + regionSize
		if addr == 0 || addr < base {
			break
		}
	}

	return matches, nil
}

func isReadable(protect uint32) bool {
	switch protect & 0xFF { // mask out modifier flags
	case windows.PAGE_READONLY,
		windows.PAGE_READWRITE,
		windows.PAGE_WRITECOPY,
		windows.PAGE_EXECUTE_READ,
		windows.PAGE_EXECUTE_READWRITE,
		windows.PAGE_EXECUTE_WRITECOPY:
		return true
	default:
		return false
	}
}

func isWritable(protect uint32) bool {
	switch protect & 0xFF {
	case windows.PAGE_READWRITE,
		windows.PAGE_WRITECOPY,
		windows.PAGE_EXECUTE_READWRITE,
		windows.PAGE_EXECUTE_WRITECOPY:
		return true
	default:
		return false
	}
}
