//go:build windows

package output

// setupResizeSignal is a no-op on Windows (SIGWINCH doesn't exist)
func (p *LivePrinter) setupResizeSignal() {
	// Windows doesn't have SIGWINCH, so we don't set up resize handling
	// Terminal resize handling would require Windows-specific APIs
}

// stopResizeSignal is a no-op on Windows
func (p *LivePrinter) stopResizeSignal() {
	// No-op on Windows
}
