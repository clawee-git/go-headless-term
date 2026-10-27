package headlessterm

func (t *Terminal) SetReflow(enabled bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.reflow = enabled
}
