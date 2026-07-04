package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Device struct {
	IP     string
	MAC    string
	Vendor string
}

// model representa el estado completo de la aplicación en un momento dado.
type model struct {
	devices  []Device
	cursor   int // índice del dispositivo actualmente seleccionado
}

func main() {
	devices := scanDevices()

	m := model{devices: devices}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

func scanDevices() []Device {
	cmd := exec.Command("sudo", "arp-scan", "--interface=enp1s0", "--localnet")
	output, _ := cmd.CombinedOutput()
	return parseArpScan(string(output))
}

func parseArpScan(output string) []Device {
	var devices []Device
	lineRegex := regexp.MustCompile(`^(\d+\.\d+\.\d+\.\d+)\s+([0-9a-fA-F:]{17})\s+(.+)$`)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		matches := lineRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		devices = append(devices, Device{
			IP:     matches[1],
			MAC:    matches[2],
			Vendor: strings.TrimSpace(matches[3]),
		})
	}
	return devices
}

// Init se ejecuta una sola vez al arrancar el programa. No necesitamos
// hacer nada acá porque el escaneo ya lo hicimos antes de lanzar la TUI.
func (m model) Init() tea.Cmd {
	return nil
}

// Update recibe cada evento (tecla presionada, etc.) y devuelve el
// modelo actualizado. Es el corazón de la arquitectura Bubble Tea.
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

// View dibuja la pantalla completa a partir del estado actual del modelo.
func (m model) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)

	b.WriteString(titleStyle.Render("netwatch — dispositivos en la red") + "\n\n")

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