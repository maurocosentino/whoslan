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

type model struct {
	store            *store.Store
	cursor           int
	err              error
	showHistory      bool
	networkInterface string
	scanInterval     time.Duration
	renaming         bool
	renameInput      textinput.Model
	t                i18n.Strings
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
	interval := flag.Duration("interval", 30*time.Second, "Intervalo entre escaneos (ej: 30s, 1m)")
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
		scanInterval:     *interval,
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

func (m model) tick() tea.Cmd {
	return tea.Tick(m.scanInterval, func(t time.Time) tea.Msg {
		return m.doScan()()
	})
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.doScan(), m.tick())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.renaming {
		return m.updateRenaming(msg)
	}

	switch msg := msg.(type) {
	case scanResultMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, m.tick()
		}
		m.err = nil
		m.store.ApplyScan(msg.devices)
		m.store.Save()
		return m, m.tick()
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.currentList())-1 {
				m.cursor++
			}
		case "h":
			m.showHistory = !m.showHistory
			m.cursor = 0
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

// onlineDevices filtra y ordena (por IP) los dispositivos actualmente online.
func onlineDevices(s *store.Store) []*store.DeviceRecord {
	var result []*store.DeviceRecord
	for _, d := range s.Devices {
		if d.Online {
			result = append(result, d)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].IP < result[j].IP
	})
	return result
}


// formatSince devuelve "hace Xh Ym" si el momento fue dentro de las
// últimas 24hs, o la fecha y hora exacta si fue antes.
func formatSince(t time.Time) string {
	elapsed := time.Since(t)
	if elapsed <= 24*time.Hour {
		return "hace " + formatDuration(elapsed)
	}
	return t.Format("02/01 15:04")
}

// currentList devuelve la lista a mostrar según la vista activa.
func (m model) currentList() []*store.DeviceRecord {
	if m.showHistory {
		return allDevices(m.store)
	}
	return onlineDevices(m.store)
}

// displayName devuelve el nombre asignado al dispositivo, o el vendor
// como fallback si todavía no tiene nombre.
func displayName(d *store.DeviceRecord) string {
	if d.Name != "" {
		return d.Name
	}
	return d.Vendor
}

// recentDevices devuelve todos los dispositivos vistos dentro de la
// ventana de tiempo indicada (online u offline), ordenados por LastSeen
// descendente (los más recientes primero).
// allDevices devuelve todos los dispositivos alguna vez vistos,
// ordenados por LastSeen descendente (los más recientes primero).
func allDevices(s *store.Store) []*store.DeviceRecord {
	var result []*store.DeviceRecord
	for _, d := range s.Devices {
		result = append(result, d)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if !result[i].LastSeen.Equal(result[j].LastSeen) {
			return result[i].LastSeen.After(result[j].LastSeen)
		}
		return result[i].MAC < result[j].MAC // desempate determinístico
	})
	return result
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
			duration = fmt.Sprintf(m.t.DisconnectedFor, formatSince(d.LastSeen))
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

	return t.View()
}

func (m model) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	title := m.t.TitleOnline
	if m.showHistory {
		title = m.t.TitleHistory
	}
	b.WriteString(titleStyle.Render(title) + "\n\n")

	if m.err != nil {
		b.WriteString(fmt.Sprintf(m.t.ScanError, m.err))
	}

	if m.renaming {
		b.WriteString(m.t.RenamePrompt + m.renameInput.View() + "\n")
		b.WriteString("\n" + dimStyle.Render(m.t.RenameHelp) + "\n")
		return b.String()
	}

	b.WriteString(m.buildTable())
	b.WriteString("\n" + dimStyle.Render(fmt.Sprintf(m.t.HelpBar, m.scanInterval)) + "\n")

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