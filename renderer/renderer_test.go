package renderer

import (
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

// BenchmarkRenderer tests worker and buffer combinations to find the best local throughput.
// To run: go test -bench=. -benchmem -benchtime=10s ./renderer
func BenchmarkRenderer(b *testing.B) {
	sourceBytes, err := os.ReadFile("file.typ")
	if err != nil {
		sourceBytes = []byte("= Benchmark Test\nHello world")
	}
	sourceCode := string(sourceBytes)

	configs := []struct {
		workers int
		buffer  int
	}{
		{workers: 4, buffer: 4},
		{workers: 4, buffer: 8},
		{workers: 4, buffer: 16},
		{workers: 6, buffer: 6},
		{workers: 6, buffer: 12},
		{workers: 6, buffer: 24},
		{workers: 8, buffer: 8},
		{workers: 8, buffer: 16},
		{workers: 8, buffer: 32},
		{workers: 12, buffer: 12},
		{workers: 12, buffer: 24},
		{workers: 12, buffer: 48},
	}

	for _, config := range configs {
		config := config
		ratio := config.buffer / config.workers
		name := fmt.Sprintf("W:%d/Ratio:%dx(Buf:%d)", config.workers, ratio, config.buffer)

		b.Run(name, func(b *testing.B) {
			r := New(RendererNew{
				InputChanSize:  config.buffer,
				OutputChanSize: config.buffer,
				Workers:        config.workers,
				ProcessTimeout: 30 * time.Second,
			})

			var drainWg sync.WaitGroup
			drainWg.Add(2)

			go func() {
				defer drainWg.Done()
				for range r.OutputChan {
				}
			}()

			go func() {
				defer drainWg.Done()
				for range r.ErrorChan {
				}
			}()

			b.SetBytes(int64(len(sourceCode)))
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				r.CreatePDF(sourceCode)
			}

			r.WaitAndClose()
			drainWg.Wait()
		})
	}
}
