package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"whoslan/internal/scanner"
)

type model struct {
	devices []scanner.Device
	cursor  int
	err     error
}

func main() {
	devices, err := scanner.Scan("enp1s0")

	m := model{devices: devices, err: err}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.devices)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)

	b.WriteString(titleStyle.Render("whoslan — dispositivos en la red") + "\n\n")

	if m.err != nil {
		b.WriteString(fmt.Sprintf("Error escaneando: %v\n", m.err))
		return b.String()
	}

	for i, d := range m.devices {
		line := fmt.Sprintf("%-16s %-18s %s", d.IP, d.MAC, d.Vendor)
		if i == m.cursor {
			b.WriteString(selectedStyle.Render("> "+line) + "\n")
		} else {
			b.WriteString("  " + line + "\n")
		}
	}

	b.WriteString("\n(↑/↓ o j/k para moverte, q para salir)\n")

	return b.String()
}