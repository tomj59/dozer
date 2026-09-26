package emu

import (
	"fmt"
	"io"
	"strings"
	"testing"
)

// flood is ~1 MB of numbered lines, like `seq`.
var flood = func() []byte {
	var b strings.Builder
	for i := 0; b.Len() < 1<<20; i++ {
		fmt.Fprintf(&b, "%d\r\n", i)
	}
	return []byte(b.String())
}()

// logLines is ~1 MB of 100-column log-like lines.
var logLines = func() []byte {
	var b strings.Builder
	for i := 0; b.Len() < 1<<20; i++ {
		fmt.Fprintf(&b, "2026-09-26T10:%02d:%02d INFO request id=%08d path=/api/v1/items status=200 dur=%dms\r\n", i/60%60, i%60, i, i%97)
	}
	return []byte(b.String())
}()

func benchEmu(b *testing.B, name string) { benchData(b, name, flood) }

func benchData(b *testing.B, name string, data []byte) {
	e, _ := New(name, 120, 40)
	go func() { _, _ = io.Copy(io.Discard, e.Replies()) }()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = e.Write(data)
	}
}

func BenchmarkFloodCharm(b *testing.B) { benchEmu(b, "charm") }
func BenchmarkFloodVT10x(b *testing.B) { benchEmu(b, "vt10x") }

func BenchmarkLogCharm(b *testing.B) { benchData(b, "charm", logLines) }
func BenchmarkLogVT10x(b *testing.B) { benchData(b, "vt10x", logLines) }
