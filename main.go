package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"

	"whoslan/internal/scanner"
	"whoslan/internal/store"
)

// const scanInterval = 30 * time.Second
// const networkInterface = "enp1s0"

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
}

func main() {
	iface := flag.String("interface", "enp1s0", "Interfaz de red a escanear (ej: enp1s0, wlan0)")
	interval := flag.Duration("interval", 30*time.Second, "Intervalo entre escaneos (ej: 30s, 1m)")
	flag.Parse()

	s, err := store.Load()
	if err != nil {
		fmt.Println("Error cargando historial:", err)
		os.Exit(1)
	}

	m := model{
		store:            s,
		networkInterface: *iface,
		scanInterval:     *interval,
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

func (m model) buildTable() string {
	columns := []table.Column{
		{Title: "Nombre", Width: 22},
		{Title: "IP", Width: 15},
		{Title: "MAC", Width: 17},
		{Title: "Estado", Width: 10},
		{Title: "Duración", Width: 18},
	}

	now := time.Now()
	devices := m.currentList()
	rows := make([]table.Row, 0, len(devices))

	for _, d := range devices {
		name := displayName(d)
		if isNewDevice(d) {
			name = "⚠️ " + name
		}

		status := "Online"
		duration := "hace " + formatDuration(now.Sub(d.FirstSeen))
		if !d.Online {
			status = "Offline"
			duration = formatSince(d.LastSeen)
		}

		rows = append(rows, table.Row{name, d.IP, d.MAC, status, duration})
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

	title := "whoslan — dispositivos online"
	if m.showHistory {
		title = "whoslan — historial completo"
	}
	b.WriteString(titleStyle.Render(title) + "\n\n")

	if m.err != nil {
		b.WriteString(fmt.Sprintf("Error escaneando: %v\n", m.err))
	}

	if m.renaming {
		b.WriteString("Nuevo nombre: " + m.renameInput.View() + "\n")
		b.WriteString("\n" + dimStyle.Render("(enter para confirmar · esc para cancelar)") + "\n")
		return b.String()
	}

	b.WriteString(m.buildTable())
	b.WriteString("\n" + dimStyle.Render(fmt.Sprintf("(↑/↓ para moverte · h para historial/online · a para reconocer · r para renombrar · q para salir · escaneo cada %s)", m.scanInterval)) + "\n")

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