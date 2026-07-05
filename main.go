package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
	"os/exec"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"

	"whoslan/internal/scanner"
	"whoslan/internal/store"
	"whoslan/internal/i18n"
)

// scanResultMsg es el mensaje que Bubble Tea recibe cuando termina un
// escaneo (disparado por tea.Tick). Encapsula tanto el resultado como
// un posible error, para que Update decida qué hacer.
type scanResultMsg struct {
	devices []scanner.Device
	err     error
}

type screen int

const (
	screenMenu screen = iota
	screenDevices
)

type model struct {
	store            *store.Store
	cursor           int
	err              error
	networkInterface string
	renaming         bool
	renameInput      textinput.Model
	t                i18n.Strings
	screen           screen
	menuCursor       int
}

// ensureSudo le pide la contraseña de sudo al usuario de forma interactiva
// ANTES de lanzar la TUI. Esto evita que sudo intente pedir la contraseña
// en background más tarde (durante un escaneo automático), lo cual
// compite por stdin con Bubble Tea y rompe los atajos de teclado.
func ensureSudo() error {
	cmd := exec.Command("sudo", "-v")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// keepSudoAlive refresca el cache de sudo cada 4 minutos en background,
// para que no expire en sesiones largas y vuelva a pedir contraseña
// mientras la TUI ya está corriendo.
func keepSudoAlive() {
	for {
		time.Sleep(4 * time.Minute)
		exec.Command("sudo", "-v").Run() // no debería pedir contraseña si el cache sigue vigente
	}
}

func main() {
	iface := flag.String("interface", "enp1s0", "Interfaz de red a escanear (ej: enp1s0, wlan0)")
	lang := flag.String("lang", "es", "Idioma de la interfaz (es, en)")
	flag.Parse()

	if err := ensureSudo(); err != nil {
		fmt.Println("Se necesita acceso sudo para escanear la red:", err)
		os.Exit(1)
	}
	go keepSudoAlive()

	s, err := store.Load()
	if err != nil {
		fmt.Println("Error cargando historial:", err)
		os.Exit(1)
	}

	m := model{
		store:            s,
		networkInterface: *iface,
		t:                i18n.Load(*lang),
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

// doScan ejecuta el escaneo en background y devuelve un tea.Cmd:
// una función que Bubble Tea ejecuta fuera del hilo principal de UI,
// y cuyo resultado llega de vuelta a Update como un mensaje.
func (m model) doScan() tea.Cmd {
	return func() tea.Msg {
		devices, err := scanner.Scan(m.networkInterface)
		return scanResultMsg{devices: devices, err: err}
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.renaming {
		return m.updateRenaming(msg)
	}

	switch m.screen {
	case screenMenu:
		return m.updateMenu(msg)
	case screenDevices:
		return m.updateDevices(msg)
	}
	return m, nil
}

// updateMenu maneja la navegación del menú principal.
func (m model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "up", "k":
		if m.menuCursor > 0 {
			m.menuCursor--
		}
	case "down", "j":
		if m.menuCursor < len(m.t.MenuItems)-1 {
			m.menuCursor++
		}
	case "enter":
		selected := m.t.MenuItems[m.menuCursor].Key
		switch selected {
		case "d":
			m.screen = screenDevices
			m.cursor = 0
			return m, m.doScan()
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

// updateDevices contiene toda la lógica que antes vivía directamente en
// Update: navegación, escaneo manual, reconocer, renombrar.
func (m model) updateDevices(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case scanResultMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.store.ApplyScan(msg.devices)
		m.store.Save()
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.screen = screenMenu
			return m, nil
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.currentList())-1 {
				m.cursor++
			}
		case "a":
			devices := m.currentList()
			if m.cursor < len(devices) {
				m.store.Acknowledge(devices[m.cursor].MAC)
				m.store.Save()
			}
		case "r":
			devices := m.currentList()
			if m.cursor < len(devices) {
				ti := textinput.New()
				ti.Placeholder = displayName(devices[m.cursor])
				ti.SetValue(devices[m.cursor].Name)
				ti.Focus()
				ti.CharLimit = 30
				m.renaming = true
				m.renameInput = ti
				return m, textinput.Blink
			}
		case "s":
			return m, m.doScan()
		}
	}
	return m, nil
}

// updateRenaming maneja los eventos mientras el usuario está escribiendo
// un nombre nuevo para el dispositivo seleccionado.
func (m model) updateRenaming(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.renaming = false
			return m, nil
		case "enter":
			devices := m.currentList()
			if m.cursor < len(devices) {
				m.store.SetName(devices[m.cursor].MAC, m.renameInput.Value())
				m.store.Save()
			}
			m.renaming = false
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.renameInput, cmd = m.renameInput.Update(msg)
	return m, cmd
}

// orderedDevices devuelve todos los dispositivos en un único orden:
// primero los online (por IP), después los offline (por LastSeen
// descendente, el que se desconectó más recientemente arriba).
func orderedDevices(s *store.Store) []*store.DeviceRecord {
	var online, offline []*store.DeviceRecord

	for _, d := range s.Devices {
		if d.Online {
			online = append(online, d)
		} else {
			offline = append(offline, d)
		}
	}

	sort.SliceStable(online, func(i, j int) bool {
		return online[i].IP < online[j].IP
	})
	sort.SliceStable(offline, func(i, j int) bool {
		if !offline[i].LastSeen.Equal(offline[j].LastSeen) {
			return offline[i].LastSeen.After(offline[j].LastSeen)
		}
		return offline[i].MAC < offline[j].MAC
	})

	return append(online, offline...)
}


// formatSince devuelve "hace Xh Ym" si el momento fue dentro de las
// últimas 24hs, o la fecha y hora exacta si fue antes.
func formatSince(t time.Time, format string) string {
	elapsed := time.Since(t)
	if elapsed <= 24*time.Hour {
		return fmt.Sprintf(format, formatDuration(elapsed))
	}
	return t.Format("02/01 15:04")
}

func (m model) currentList() []*store.DeviceRecord {
	return orderedDevices(m.store)
}

// displayName devuelve el nombre asignado al dispositivo, o el vendor
// como fallback si todavía no tiene nombre.
func displayName(d *store.DeviceRecord) string {
	if d.Name != "" {
		return d.Name
	}
	return d.Vendor
}

// formatDuration convierte una duración a un texto legible tipo "2h 15m".
func formatDuration(d time.Duration) string {
	d = d.Round(time.Minute)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

// columnWidth calcula el ancho necesario para una columna, en base al
// título y al contenido más largo, respetando un mínimo y un máximo.
func columnWidth(title string, values []string, min, max int) int {
	width := len(title)
	for _, v := range values {
		if len(v) > width {
			width = len(v)
		}
	}
	if width < min {
		width = min
	}
	if width > max {
		width = max
	}
	return width
}

func (m model) buildTable() string {
	now := time.Now()
	devices := m.currentList()

	names := make([]string, len(devices))
	ips := make([]string, len(devices))
	macs := make([]string, len(devices))
	statuses := make([]string, len(devices))
	durations := make([]string, len(devices))

	rows := make([]table.Row, 0, len(devices))

	for i, d := range devices {
		name := displayName(d)

		alert := " "
		if isNewDevice(d) {
			alert = "!"
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
	return dimOfflineRows(rendered, devices, m.cursor)
}

func (m model) viewMenu() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	subtitleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	b.WriteString(titleStyle.Render(m.t.AppTitle) + "\n")
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

	return b.String()
}

func (m model) View() string {
	if m.screen == screenMenu {
		return m.viewMenu()
	}
	return m.viewDevices()
}

// viewDevices es tu View() original, renombrado.
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
		return b.String()
	}

	b.WriteString(m.buildTable())
	b.WriteString("\n" + buildHelpBar(m.t.HelpItems, dimStyle, keyStyle) + "\n")

	return b.String()
}

// isUnknownVendor detecta MACs con randomización/administración local,
// que arp-scan reporta con este texto característico.
func isUnknownVendor(vendor string) bool {
	return strings.Contains(vendor, "Unknown: locally administered")
}

// isNewDevice indica si un dispositivo todavía no fue reconocido
// manualmente por el usuario (tecla "a").
func isNewDevice(d *store.DeviceRecord) bool {
	return !d.Acknowledged
}

// dimOfflineRows recorre el texto ya renderizado por la tabla línea por
// línea, y pinta en gris las filas de dispositivos offline (salvo la
// fila actualmente seleccionada, que ya tiene su propio resaltado).
func dimOfflineRows(rendered string, devices []*store.DeviceRecord, cursor int) string {
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	lines := strings.Split(rendered, "\n")

	// La línea 0 es el header; las filas de datos arrancan en la línea 1.
	for i, d := range devices {
		lineIdx := i + 1
		if lineIdx >= len(lines) {
			break
		}
		if !d.Online && i != cursor {
			lines[lineIdx] = dimStyle.Render(lines[lineIdx])
		}
	}

	return strings.Join(lines, "\n")
}

// buildHelpBar arma la barra de ayuda, resaltando en negrita solo la
// tecla de cada atajo (no la descripción completa).
func buildHelpBar(items []i18n.HelpItem, dimStyle, keyStyle lipgloss.Style) string {
	var parts []string
	for _, item := range items {
		parts = append(parts, keyStyle.Render(item.Key)+dimStyle.Render(" "+item.Action))
	}
	return dimStyle.Render("(") + strings.Join(parts, dimStyle.Render(" · ")) + dimStyle.Render(")")
}