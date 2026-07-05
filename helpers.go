package main

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"whoslan/internal/i18n"
	"whoslan/internal/store"
)

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

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
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

// highlightAlertColumn recorre el texto ya renderizado por la tabla y
// pinta de color el símbolo "!" en la primera columna, salvo en la fila
// actualmente seleccionada (que ya tiene su propio resaltado de cursor,
// y mezclar los dos estilos rompe el color de la selección).
func highlightAlertColumn(rendered string, alerts []bool, cursor int) string {
	lines := strings.Split(rendered, "\n")

	for i, isAlert := range alerts {
		if !isAlert || i == cursor {
			continue
		}
		lineIdx := i + 1
		if lineIdx >= len(lines) {
			continue
		}
		line := lines[lineIdx]
		idx := strings.Index(line, "!")
		if idx == -1 {
			continue
		}
		lines[lineIdx] = line[:idx] + alertStyle.Render("!") + line[idx+1:]
	}

	return strings.Join(lines, "\n")
}