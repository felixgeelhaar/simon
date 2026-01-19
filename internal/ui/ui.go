package ui

type UI interface {
	UpdateStatus(status string)
	UpdateIteration(iter int)
	Log(msg string)
}

type SilentUI struct{}
func (s SilentUI) UpdateStatus(status string) {}
func (s SilentUI) UpdateIteration(iter int)   {}
func (s SilentUI) Log(msg string)             {}
