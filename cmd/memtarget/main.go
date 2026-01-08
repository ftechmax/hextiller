package main

import (
	"bufio"
	"fmt"
	"os"
	"sync"
	"time"
)

type trackedValues struct {
	Int32   int32
	Uint32  uint32
	Int64   int64
	Uint64  uint64
	Float32 float32
	Float64 float64
	Bool    bool
	Message string
	Bytes   [8]byte
}

var state = &trackedValues{
	Int32:   1337,
	Uint32:  421337,
	Int64:   4200,
	Uint64:  9001,
	Float32: 3.14,
	Float64: 12.5,
	Bool:    true,
	Message: "hello gomem",
	Bytes:   [8]byte{1, 2, 3, 4, 5, 6, 7, 8},
}

var mu sync.Mutex

const refresh = 500 * time.Millisecond

func main() {
	ticker := time.NewTicker(refresh)
	defer ticker.Stop()

	go listenInput()

	for range ticker.C {
		render()
	}
}

func listenInput() {
	reader := bufio.NewReader(os.Stdin)
	for {
		ch, err := reader.ReadByte()
		if err != nil {
			return
		}
		switch ch {
		case '+':
			mu.Lock()
			state.Int32++
			state.Int64++
			state.Uint32++
			state.Uint64++
			state.Float32 += 1
			state.Float64 += 1
			mu.Unlock()
		case '-':
			mu.Lock()
			if state.Uint64 > 0 {
				state.Uint64--
			}
			if state.Uint32 > 0 {
				state.Uint32--
			}
			state.Int32--
			state.Int64--
			state.Float32 -= 1
			state.Float64 -= 1
			mu.Unlock()
		default:
			// ignore other keys
		}
	}
}

func render() {
	mu.Lock()
	vals := *state
	mu.Unlock()

	fmt.Print("\033[H\033[2J") // clear screen for refreshed view
	fmt.Println("gomem memory test target (Ctrl+C to exit; +/- to change values, press Enter after key on Windows)")
	fmt.Println()
	fmt.Printf("int32:    %d\n", vals.Int32)
	fmt.Printf("uint32:   %d\n", vals.Uint32)
	fmt.Printf("int64:    %d\n", vals.Int64)
	fmt.Printf("uint64:   %d\n", vals.Uint64)
	fmt.Printf("float32:  %.2f\n", vals.Float32)
	fmt.Printf("float64:  %.4f\n", vals.Float64)
	fmt.Printf("bool:     %t\n", vals.Bool)
	fmt.Printf("bytes:    %v\n", vals.Bytes)
	fmt.Printf("message:  %s\n", vals.Message)
}
