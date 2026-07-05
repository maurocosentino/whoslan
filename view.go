package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/common-nighthawk/go-figure"
	"whoslan/internal/portscan"
	"unicode/utf8"
)

func (m model) View() string {
	switch m.screen {
	case screenMenu:
		return m.viewMenu()
	case screenPorts:
		return m.viewPorts()
	case screenConnections:
		return m.viewConnections()
	case screenInterface:
		return m.viewInterface()
	default:
		return m.viewDevices()
	}
}

// alertStyle es el estilo compartido para el símbolo de alerta "!"
// que indica un dispositivo o puerto nuevo sin reconocer.
var alertStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("214"))

// asciiTitle genera el título de la app como arte ASCII grande.
func asciiTitle(text string) string {
	fig := figure.NewFigure(text, "standard", true)
	return fig.String()
}

func (m model) viewMenu() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	subtitleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	b.WriteString(titleStyle.Render(asciiTitle(m.t.AppTitle)) + "\n")
	b.WriteString(subtitleStyle.Render(m.t.AppSubtitle) + "\n\n")

	for i, item := range m.t.MenuItems {
		line := fmt.Sprintf("[%s] %-15s %s", item.Key, item.Label, item.Description)
		if i == m.menuCursor {
			b.WriteString(selectedStyle.Render("→ "+line) + "\n")
		} else {
			b.WriteString("  " + descStyle.Render(line) + "\n")
		}
	}

	b.WriteString("\n" + subtitleStyle.Render(m.t.MenuHelp) + "\n")

	content := b.String()
	return m.centered(content)
}

// centered envuelve el contenido para que aparezca centrado en la
// terminal, usando el tamaño real reportado por Bubble Tea. Si todavía
// no lo conocemos (arranque muy temprano), devuelve el contenido tal cual.
func (m model) centered(content string) string {
	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
	}
	return content
}

func (m model) viewDevices() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("240"))

	b.WriteString(titleStyle.Render(m.t.Title) + "\n\n")

	if m.err != nil {
		b.WriteString(fmt.Sprintf(m.t.ScanError, m.err))
	}

	if m.renaming {
		b.WriteString(m.t.RenamePrompt + m.renameInput.View() + "\n")
		b.WriteString("\n" + dimStyle.Render(m.t.RenameHelp) + "\n")
		return m.centered(b.String())
	}

	b.WriteString(m.buildTable())
	b.WriteString("\n" + buildHelpBar(m.t.HelpItems, dimStyle, keyStyle) + "\n")

	return m.centered(b.String())
}

func (m model) buildTable() string {
	now := time.Now()
	devices := m.currentList()

	names := make([]string, len(devices))
	ips := make([]string, len(devices))
	macs := make([]string, len(devices))
	statuses := make([]string, len(devices))
	durations := make([]string, len(devices))
	alerts := make([]bool, len(devices))

	rows := make([]table.Row, 0, len(devices))

	for i, d := range devices {
		name := displayName(d)

		alert := " "
		if isNewDevice(d) {
			alert = "!"
			alerts[i] = true
		}

		status := m.t.StatusOnline
		duration := fmt.Sprintf(m.t.ConnectedFor, formatDuration(now.Sub(d.FirstSeen)))
		if !d.Online {
			status = m.t.StatusOffline
			duration = formatSince(d.LastSeen, m.t.ConnectedFor)
		}
		names[i] = name
		ips[i] = d.IP
		macs[i] = d.MAC
		statuses[i] = status
		durations[i] = duration

		rows = append(rows, table.Row{alert, name, d.IP, d.MAC, status, duration})
	}

	columns := []table.Column{
		{Title: m.t.ColAlert, Width: 1},
		{Title: m.t.ColName, Width: columnWidth(m.t.ColName, names, 10, 40)},
		{Title: m.t.ColIP, Width: columnWidth(m.t.ColIP, ips, 10, 15)},
		{Title: m.t.ColMAC, Width: columnWidth(m.t.ColMAC, macs, 10, 17)},
		{Title: m.t.ColStatus, Width: columnWidth(m.t.ColStatus, statuses, 6, 10)},
		{Title: m.t.ColDuration, Width: columnWidth(m.t.ColDuration, durations, 8, 20)},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(len(rows)+1),
	)
	t.SetCursor(m.cursor)

	rendered := t.View()
	rendered = highlightAlertColumn(rendered, alerts, m.cursor)
	return dimOfflineRows(rendered, devices, m.cursor)
}

func (m model) viewPorts() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("240"))

	b.WriteString(titleStyle.Render(m.t.PortsTitle) + "\n\n")

	if m.err != nil {
		b.WriteString(fmt.Sprintf(m.t.PortsError, m.err))
	}

	b.WriteString(m.buildPortsTable())
	b.WriteString("\n" + buildHelpBar(m.t.PortsHelp, dimStyle, keyStyle) + "\n")

	return m.centered(b.String())
}

func (m model) buildPortsTable() string {
	columns := []table.Column{
		{Title: m.t.ColAlert, Width: 1},
		{Title: m.t.ColPort, Width: 8},
		{Title: m.t.ColProtocol, Width: 10},
		{Title: m.t.ColProcess, Width: 25},
		{Title: m.t.ColPID, Width: 8},
	}

	rows := make([]table.Row, 0, len(m.ports))
	alerts := make([]bool, len(m.ports))

	for i, p := range m.ports {
		alert := " "
		key := fmt.Sprintf("%s:%d", p.Protocol, p.Port)
		if record, exists := m.store.Ports[key]; exists && !record.Acknowledged {
			alert = "!"
			alerts[i] = true
		}

		rows = append(rows, table.Row{
			alert,
			fmt.Sprintf("%d", p.Port),
			p.Protocol,
			p.ProcessName,
			fmt.Sprintf("%d", p.PID),
		})
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(len(rows)+1),
	)
	t.SetCursor(m.portsCursor)
	return highlightAlertColumn(t.View(), alerts, m.portsCursor)
}

func (m model) viewConnections() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("240"))

	b.WriteString(titleStyle.Render(m.t.ConnTitle) + "\n\n")

	if m.err != nil {
		b.WriteString(fmt.Sprintf(m.t.ConnError, m.err))
	}

	b.WriteString(m.buildConnectionsTable())
	b.WriteString("\n" + buildHelpBar(m.t.ConnHelp, dimStyle, keyStyle) + "\n")

	return m.centered(b.String())
}

func (m model) buildConnectionsTable() string {
	columns := []table.Column{
		{Title: m.t.ColProtocol, Width: 10},
		{Title: m.t.ColLocal, Width: 22},
		{Title: m.t.ColRemote, Width: 22},
		{Title: m.t.ColCountry, Width: 8},
		{Title: m.t.ColConnStatus, Width: 14},
		{Title: m.t.ColProcess, Width: 20},
	}

	rows := make([]table.Row, 0, len(m.connections))
	for _, c := range m.connections {
		ip := remoteIP(c.RemoteAddr)

		country, cached := m.store.CountryFor(ip)
		if !cached {
			country = portscan.LookupCountry(ip)
			m.store.SetCountry(ip, country)
		}

		if country == "" {
			country = "-"
		}
		rows = append(rows, table.Row{c.Protocol, c.LocalAddr, c.RemoteAddr, country, c.Status, c.ProcessName})
	}

	m.store.SaveGeoCache()

	t := table.New(
		table.WithColumns(columns),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithHeight(len(rows)+1),
	)
	t.SetCursor(m.connCursor)

	return t.View()
}

// remoteIP extrae solo la IP (sin puerto) de un string "IP:puerto".
func remoteIP(addr string) string {
	idx := strings.LastIndex(addr, ":")
	if idx == -1 {
		return addr
	}
	return addr[:idx]
}

func (m model) viewInterface() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	keyStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("240"))

	b.WriteString(titleStyle.Render(m.t.InterfaceTitle) + "\n\n")

	if m.err != nil {
		b.WriteString(fmt.Sprintf(m.t.InterfaceError, m.err) + "\n\n")
	}

	na := m.t.NotAvailable
	fields := []struct {
		label string
		value string
	}{
		{m.t.LabelInterface, orDefault(m.ifaceInfo.Name, na)},
		{m.t.LabelLocalIP, orDefault(m.ifaceInfo.LocalIP, na)},
		{m.t.LabelNetmask, orDefault(m.ifaceInfo.Netmask, na)},
		{m.t.LabelGateway, orDefault(m.ifaceInfo.Gateway, na)},
		{m.t.LabelPublicIP, orDefault(m.ifaceInfo.PublicIP, na)},
	}

	// Ancho de etiqueta uniforme, contando caracteres (runas) y no bytes,
	// para que acentos como "á" en español no rompan la alineación.
	labelWidth := 0
	for _, f := range fields {
		w := utf8.RuneCountInString(f.label) + 1 // +1 por los ":"
		if w > labelWidth {
			labelWidth = w
		}
	}

	// Ancho total de línea (etiqueta + separador + valor), para saber
	// cuánto rellenar antes de centrar el bloque completo.
	lineWidth := 0
	for _, f := range fields {
		w := labelWidth + 1 + utf8.RuneCountInString(f.value)
		if w > lineWidth {
			lineWidth = w
		}
	}

	for _, f := range fields {
		label := f.label + ":"
		labelPad := labelWidth - utf8.RuneCountInString(label)
		line := label + strings.Repeat(" ", labelPad) + " " + f.value

		linePad := lineWidth - utf8.RuneCountInString(line)
		line += strings.Repeat(" ", linePad)

		b.WriteString(labelStyle.Render(label+strings.Repeat(" ", labelPad)) + " " + f.value + strings.Repeat(" ", linePad) + "\n")
	}

	b.WriteString("\n" + buildHelpBar(m.t.InterfaceHelp, dimStyle, keyStyle) + "\n")

	return m.centered(b.String())
}
