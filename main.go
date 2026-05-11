package main

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/iamharshdabas/typst-concurrent-renderer/renderer"
)

func main() {
	sourceBytes, err := os.ReadFile("renderer/file.typ")
	if err != nil {
		fmt.Printf("Fatal: Could not read source file: %v\n", err)
		return
	}
	sourceCode := string(sourceBytes)

	totalRuns := 1000
	workers := runtime.NumCPU()
	bufferSize := workers * 2

	r := renderer.New(renderer.RendererNew{
		InputChanSize:  bufferSize,
		OutputChanSize: bufferSize,
		Workers:        workers,
	})

	var successCount, errorCount int
	var listenerWg sync.WaitGroup
	listenerWg.Add(2)

	go func() {
		defer listenerWg.Done()
		for range r.OutputChan {
			successCount++
		}
	}()

	go func() {
		defer listenerWg.Done()
		for errVal := range r.ErrorChan {
			errorCount++
			if errorCount <= 5 {
				fmt.Printf("[Error] %v\n", errVal)
			}
		}
	}()

	fmt.Printf("Starting stress test: %d runs | %d workers | %d buffer\n", totalRuns, workers, bufferSize)
	startTime := time.Now()

	go func() {
		for range totalRuns {
			r.CreatePDF(sourceCode)
		}

		r.WaitAndClose()
	}()

	listenerWg.Wait()

	duration := time.Since(startTime)

	fmt.Println("=====================================")
	fmt.Printf("Total Time taken : %v\n", duration)
	fmt.Printf("Avg Time taken   : %.2f ms\n", float64(duration.Milliseconds())/float64(successCount))
	fmt.Printf("Success Rate     : %d/%d\n", successCount, totalRuns)
	fmt.Printf("Error Rate       : %d/%d\n", errorCount, totalRuns)

	if duration.Seconds() > 0 {
		pdfsPerSecond := float64(successCount) / duration.Seconds()
		fmt.Printf("Throughput       : %.2f PDFs/sec\n", pdfsPerSecond)
	}
	fmt.Println("=====================================")
}
