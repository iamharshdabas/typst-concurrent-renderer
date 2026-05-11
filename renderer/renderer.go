package renderer

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type Renderer struct {
	wg             sync.WaitGroup
	inputChan      chan string
	OutputChan     chan Pdf
	ErrorChan      chan error
	processTimeout time.Duration
}

type RendererNew struct {
	InputChanSize  int
	OutputChanSize int
	Workers        int
	ProcessTimeout time.Duration
}

type Pdf []byte

func New(obj RendererNew) *Renderer {
	processTimeout := obj.ProcessTimeout
	if processTimeout == 0 {
		processTimeout = 5 * time.Second
	}

	r := &Renderer{
		inputChan:      make(chan string, obj.InputChanSize),
		OutputChan:     make(chan Pdf, obj.OutputChanSize),
		ErrorChan:      make(chan error, obj.OutputChanSize),
		processTimeout: processTimeout,
	}

	for i := 0; i < obj.Workers; i++ {
		go r.startInternalWorker()
	}

	return r
}

func (r *Renderer) CreatePDF(s string) {
	r.wg.Add(1)
	r.inputChan <- s
}

func (r *Renderer) startInternalWorker() {
	for source := range r.inputChan {
		pdfBinary, err := r.processTypst(source)
		if err != nil {
			r.ErrorChan <- err
		} else {
			r.OutputChan <- pdfBinary
		}
		r.wg.Done()
	}
}

func (r *Renderer) WaitAndClose() {
	close(r.inputChan)
	r.wg.Wait()
	close(r.OutputChan)
	close(r.ErrorChan)
}

func (r *Renderer) processTypst(s string) (Pdf, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.processTimeout)
	defer cancel()

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "typst", "compile", "-", "-")
	cmd.Stdin = strings.NewReader(s)
	cmd.Stderr = &stderr

	out, err := cmd.Output()
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("typst timed out after %s: %w", r.processTimeout, ctx.Err())
		}
		return nil, fmt.Errorf("typst err: %v, stderr: %s", err, stderr.String())
	}

	return out, nil
}
