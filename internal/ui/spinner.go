package ui

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/fatih/color"
)

type Spinner struct {
	label   string
	done    chan struct{}
	wg      sync.WaitGroup
	mu      sync.Mutex
	percent int // -1 = no percentage
}

// Start shows an animated spinner with the given label.
func Start(label string) *Spinner {
	sp := &Spinner{
		label:   label,
		done:    make(chan struct{}),
		percent: -1,
	}

	sp.wg.Add(1)
	go sp.run()

	return sp
}

// SetProgress updates the percentage shown next to the spinner (0-100).
func (s *Spinner) SetProgress(percent int) {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	s.mu.Lock()
	s.percent = percent
	s.mu.Unlock()
}

func (s *Spinner) run() {
	defer s.wg.Done()

	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	i := 0
	ticker := time.NewTicker(80 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			frame := frames[i%len(frames)]
			i++

			s.mu.Lock()
			percent := s.percent
			s.mu.Unlock()

			if percent >= 0 {
				fmt.Fprintf(os.Stderr, "\r%s %s (%d%%)", color.YellowString(frame), s.label, percent)
				continue
			}

			fmt.Fprintf(os.Stderr, "\r%s %s", color.YellowString(frame), s.label)
		}
	}
}

// Stop clears the spinner line and optionally prints a success message.
func (s *Spinner) Stop(success bool, message string) {
	close(s.done)
	s.wg.Wait()

	fmt.Fprint(os.Stderr, "\r\033[K")

	if message == "" {
		return
	}

	if success {
		color.Green("✓ %s", message)
		return
	}

	color.Red("✗ %s", message)
}

// Run runs fn while showing a spinner, then prints a success message.
func Run(label, successMessage string, fn func() error) error {
	return RunWithProgress(label, successMessage, func(setProgress func(int)) error {
		return fn()
	})
}

// RunWithProgress runs fn while showing a spinner; fn can call setProgress(0-100).
func RunWithProgress(label, successMessage string, fn func(setProgress func(int)) error) error {
	sp := Start(label)
	setProgress := func(percent int) {
		sp.SetProgress(percent)
	}

	err := fn(setProgress)
	if err != nil {
		sp.Stop(false, "")
		return err
	}

	sp.Stop(true, successMessage)
	return nil
}
