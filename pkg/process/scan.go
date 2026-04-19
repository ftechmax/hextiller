package process

import (
	"encoding/binary"
	"math"
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
	return p.scanNumeric(4, func(b []byte) bool {
		v := math.Float32frombits(binary.LittleEndian.Uint32(b))
		return float32Abs(v-target) <= eps
	}, maxResults, writableOnly)
}

func (p *Process) ScanFloat64Approx(target float64, eps float64, maxResults int, writableOnly bool) ([]uintptr, error) {
	return p.scanNumeric(8, func(b []byte) bool {
		v := math.Float64frombits(binary.LittleEndian.Uint64(b))
		return math.Abs(v-target) <= eps
	}, maxResults, writableOnly)
}

func float32Abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
