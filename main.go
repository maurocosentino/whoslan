package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"whoslan/internal/scanner"
	"whoslan/internal/store"
)

const scanInterval = 30 * time.Second
const networkInterface = "enp1s0"

// scanResultMsg es el mensaje que Bubble Tea recibe cuando termina un
// escaneo (disparado por tea.Tick). Encapsula tanto el resultado como
// un posible error, para que Update decida qué hacer.
type scanResultMsg struct {
	devices []scanner.Device
	err     error
}

type model struct {
	store      *store.Store
	cursor     int
	err        error
	showHistory bool
}

func main() {
	s, err := store.Load()
	if err != nil {
		fmt.Println("Error cargando historial:", err)
		os.Exit(1)
	}

	m := model{store: s}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}

// doScan ejecuta el escaneo en background y devuelve un tea.Cmd:
// una función que Bubble Tea ejecuta fuera del hilo principal de UI,
// y cuyo resultado llega de vuelta a Update como un mensaje.
func doScan() tea.Cmd {
	return func() tea.Msg {
		devices, err := scanner.Scan(networkInterface)
		return scanResultMsg{devices: devices, err: err}
	}
}

// tick devuelve un comando que espera el intervalo definido y luego
// dispara un nuevo escaneo. Es lo que hace que el escaneo se repita solo.
func tick() tea.Cmd {
	return tea.Tick(scanInterval, func(t time.Time) tea.Msg {
		return doScan()()
	})
}

func (m model) Init() tea.Cmd {
	// Al arrancar: un escaneo inmediato, y programamos el siguiente.
	return tea.Batch(doScan(), tick())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case scanResultMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, tick()
		}
		m.err = nil
		m.store.ApplyScan(msg.devices)
		m.store.Save() // persistimos en cada escaneo
		return m, tick()
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
		}
	}
	return m, nil
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

func (m model) View() string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true)
	offlineStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	title := "whoslan — dispositivos online"
	if m.showHistory {
		title = "whoslan — historial completo"
	}	
	b.WriteString(titleStyle.Render(title) + "\n\n")

	if m.err != nil {
		b.WriteString(fmt.Sprintf("Error escaneando: %v\n", m.err))
	}

	now := time.Now()
	devices := m.currentList()

	for i, d := range devices {
		var status string
		if d.Online {
			status = fmt.Sprintf("conectado hace %s", formatDuration(now.Sub(d.FirstSeen)))
		} else {
			status = "desconectado " + formatSince(d.LastSeen)
		}

		line := fmt.Sprintf("%-16s %-18s %-30s %s", d.IP, d.MAC, d.Vendor, status)

		switch {
		case i == m.cursor:
			b.WriteString(selectedStyle.Render("> "+line) + "\n")
		case !d.Online:
			b.WriteString(offlineStyle.Render("  "+line) + "\n")
		default:
			b.WriteString("  " + line + "\n")
		}
	}

	b.WriteString("\n" + dimStyle.Render("(↑/↓ para moverte · h para historial/online · q para salir)") + "\n")

	return b.String()
}