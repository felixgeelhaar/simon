package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TUI struct {
	program *tea.Program
}

func NewTUI(p *tea.Program) *TUI {
	return &TUI{program: p}
}

func (t *TUI) UpdateStatus(status string) {
	t.program.Send(StatusMsg(status))
}

func (t *TUI) UpdateIteration(iter int) {
	t.program.Send(IterMsg(iter))
}

func (t *TUI) Log(msg string) {
	t.program.Send(LogMsg(msg))
}

var (
	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1)

	infoStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575"))

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF0000"))
)

type Model struct {
	Title      string
	Status     string
	Iteration  int
	MaxIter    int
	Log        []string
	Progress   progress.Model
	Viewport   viewport.Model
	Quitting   bool
	Ready      bool
	Width      int
	Height     int
}

type LogMsg string
type StatusMsg string
type IterMsg int

func NewModel(title string, maxIter int) Model {
	p := progress.New(progress.WithDefaultGradient())
	return Model{
		Title:    title,
		Status:   "Initializing...",
		MaxIter:  maxIter,
		Progress: p,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC || msg.String() == "q" {
			m.Quitting = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		if !m.Ready {
			m.Viewport = viewport.New(msg.Width, msg.Height-10)
			m.Ready = true
		} else {
			m.Viewport.Width = msg.Width
			m.Viewport.Height = msg.Height - 10
		}

	case LogMsg:
		m.Log = append(m.Log, string(msg))
		m.Viewport.SetContent(strings.Join(m.Log, "\n"))
		m.Viewport.GotoBottom()

	case StatusMsg:
		m.Status = string(msg)

	case IterMsg:
		m.Iteration = int(msg)
	}

	var cmd tea.Cmd
	m.Viewport, cmd = m.Viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if !m.Ready {
		return "\n  Initializing..."
	}

	header := titleStyle.Render(" Simon AI Agent Governance ")
	status := infoStyle.Render(fmt.Sprintf(" Status: %s ", m.Status))
	iter := fmt.Sprintf(" Iteration: %d/%d ", m.Iteration, m.MaxIter)
	
	prog := m.Progress.ViewAs(float64(m.Iteration) / float64(m.MaxIter))

	view := fmt.Sprintf("%s%s%s\n\n%s\n\n%s", 
		header, status, iter,
		m.Viewport.View(),
		prog)

	if m.Quitting {
		return view + "\n  Quitting...\n"
	}

	return view
}
