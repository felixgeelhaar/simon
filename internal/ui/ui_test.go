package ui

import (
	"testing"
)

func TestSilentUI_UpdateStatus(t *testing.T) {
	ui := SilentUI{}
	// Should not panic
	ui.UpdateStatus("test status")
}

func TestSilentUI_UpdateIteration(t *testing.T) {
	ui := SilentUI{}
	// Should not panic
	ui.UpdateIteration(1)
	ui.UpdateIteration(100)
	ui.UpdateIteration(0)
}

func TestSilentUI_Log(t *testing.T) {
	ui := SilentUI{}
	// Should not panic
	ui.Log("test message")
	ui.Log("")
}

func TestSilentUI_ImplementsInterface(t *testing.T) {
	// Verify SilentUI implements UI interface
	var _ UI = SilentUI{}
	var _ UI = &SilentUI{}
}

// MockUI implements UI interface for testing
type MockUI struct {
	StatusUpdates    []string
	IterationUpdates []int
	LogMessages      []string
}

func (m *MockUI) UpdateStatus(status string) {
	m.StatusUpdates = append(m.StatusUpdates, status)
}

func (m *MockUI) UpdateIteration(iter int) {
	m.IterationUpdates = append(m.IterationUpdates, iter)
}

func (m *MockUI) Log(msg string) {
	m.LogMessages = append(m.LogMessages, msg)
}

func TestMockUI_UpdateStatus(t *testing.T) {
	ui := &MockUI{}

	ui.UpdateStatus("status1")
	ui.UpdateStatus("status2")

	if len(ui.StatusUpdates) != 2 {
		t.Errorf("expected 2 status updates, got %d", len(ui.StatusUpdates))
	}
	if ui.StatusUpdates[0] != "status1" {
		t.Errorf("expected 'status1', got %q", ui.StatusUpdates[0])
	}
	if ui.StatusUpdates[1] != "status2" {
		t.Errorf("expected 'status2', got %q", ui.StatusUpdates[1])
	}
}

func TestMockUI_UpdateIteration(t *testing.T) {
	ui := &MockUI{}

	ui.UpdateIteration(1)
	ui.UpdateIteration(2)
	ui.UpdateIteration(3)

	if len(ui.IterationUpdates) != 3 {
		t.Errorf("expected 3 iteration updates, got %d", len(ui.IterationUpdates))
	}
	for i, expected := range []int{1, 2, 3} {
		if ui.IterationUpdates[i] != expected {
			t.Errorf("expected iteration %d, got %d", expected, ui.IterationUpdates[i])
		}
	}
}

func TestMockUI_Log(t *testing.T) {
	ui := &MockUI{}

	ui.Log("message1")
	ui.Log("message2")

	if len(ui.LogMessages) != 2 {
		t.Errorf("expected 2 log messages, got %d", len(ui.LogMessages))
	}
	if ui.LogMessages[0] != "message1" {
		t.Errorf("expected 'message1', got %q", ui.LogMessages[0])
	}
}

func TestMockUI_ImplementsInterface(t *testing.T) {
	// Verify MockUI implements UI interface
	var _ UI = &MockUI{}
}

func TestUI_InterfaceMethods(t *testing.T) {
	// Test that the UI interface can be used polymorphically
	uis := []UI{
		SilentUI{},
		&MockUI{},
	}

	for _, ui := range uis {
		// These should all work without panic
		ui.UpdateStatus("test")
		ui.UpdateIteration(1)
		ui.Log("test")
	}
}
