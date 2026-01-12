//go:build unix

package output

import (
	"os"
	"os/signal"
	"syscall"
)

// setupResizeSignal sets up terminal resize signal handling on Unix systems
func (p *LivePrinter) setupResizeSignal() {
	if p.fullscreen {
		p.resizeCh = make(chan os.Signal, 1)
		signal.Notify(p.resizeCh, syscall.SIGWINCH)
	}
}

// stopResizeSignal stops the resize signal handler
func (p *LivePrinter) stopResizeSignal() {
	if p.resizeCh != nil {
		signal.Stop(p.resizeCh)
	}
}
