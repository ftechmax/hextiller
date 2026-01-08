//go:build windows

package process

import (
	"encoding/binary"
	"errors"
	"math"
	"unsafe"

	"golang.org/x/sys/windows"
)

func (p *Process) ScanInt32(target int32, maxResults int, writableOnly bool) ([]uintptr, error) {
	return p.scanNumeric(4, func(b []byte) bool {
		return int32(binary.LittleEndian.Uint32(b)) == target
	}, maxResults, writableOnly)
}

func (p *Process) ScanUint32(target uint32, maxResults int, writableOnly bool) ([]uintptr, error) {
	return p.scanNumeric(4, func(b []byte) bool {
		return binary.LittleEndian.Uint32(b) == target
	}, maxResults, writableOnly)
}

func (p *Process) ScanInt64(target int64, maxResults int, writableOnly bool) ([]uintptr, error) {
	return p.scanNumeric(8, func(b []byte) bool {
		return int64(binary.LittleEndian.Uint64(b)) == target
	}, maxResults, writableOnly)
}

func (p *Process) ScanUint64(target uint64, maxResults int, writableOnly bool) ([]uintptr, error) {
	return p.scanNumeric(8, func(b []byte) bool {
		return binary.LittleEndian.Uint64(b) == target
	}, maxResults, writableOnly)
}

func (p *Process) ScanFloat32(target float32, maxResults int, writableOnly bool) ([]uintptr, error) {
	tbits := math.Float32bits(target)
	return p.scanNumeric(4, func(b []byte) bool {
		return binary.LittleEndian.Uint32(b) == tbits
	}, maxResults, writableOnly)
}

func (p *Process) ScanFloat64(target float64, maxResults int, writableOnly bool) ([]uintptr, error) {
	tbits := math.Float64bits(target)
	return p.scanNumeric(8, func(b []byte) bool {
		return binary.LittleEndian.Uint64(b) == tbits
	}, maxResults, writableOnly)
}

func (p *Process) ScanFloat32Approx(target float32, eps float32, maxResults int, writableOnly bool) ([]uintptr, error) {
	targetF := float32(target)
	return p.scanNumeric(4, func(b []byte) bool {
		v := math.Float32frombits(binary.LittleEndian.Uint32(b))
		return float32Abs(v-targetF) <= eps
	}, maxResults, writableOnly)
}

func (p *Process) ScanFloat64Approx(target float64, eps float64, maxResults int, writableOnly bool) ([]uintptr, error) {
	targetF := target
	return p.scanNumeric(8, func(b []byte) bool {
		v := math.Float64frombits(binary.LittleEndian.Uint64(b))
		return math.Abs(v-targetF) <= eps
	}, maxResults, writableOnly)
}

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

func float32Abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
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
