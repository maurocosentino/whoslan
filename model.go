package main

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/maurocosentino/whoslan/internal/i18n"
	"github.com/maurocosentino/whoslan/internal/portscan"
	"github.com/maurocosentino/whoslan/internal/scanner"
	"github.com/maurocosentino/whoslan/internal/store"
)

// scanResultMsg es el mensaje que Bubble Tea recibe cuando termina un
// escaneo (disparado por tea.Tick). Encapsula tanto el resultado como
// un posible error, para que Update decida qué hacer.
type scanResultMsg struct {
	devices []scanner.Device
	err     error
}

type portScanResultMsg struct {
	ports []portscan.ListeningPort
	err   error
}

type connScanResultMsg struct {
	connections []portscan.Connection
	err         error
}

type ifaceInfoMsg struct {
	info portscan.InterfaceInfo
	err  error
}

type screen int

const (
	screenMenu screen = iota
	screenDevices
	screenPorts
	screenConnections
	screenInterface
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
	lang             string
	menuCursor       int
	ports            []portscan.ListeningPort
	portsCursor      int
	connections      []portscan.Connection
	connCursor       int
	ifaceInfo        portscan.InterfaceInfo
	width 			 int
	height			 int
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

func (m model) doPortScan() tea.Cmd {
	return func() tea.Msg {
		ports, err := portscan.Scan()
		return portScanResultMsg{ports: ports, err: err}
	}
}

func (m model) doConnScan() tea.Cmd {
	return func() tea.Msg {
		conns, err := portscan.ScanConnections()
		return connScanResultMsg{connections: conns, err: err}
	}
}

func (m model) doGetInterfaceInfo() tea.Cmd {
	return func() tea.Msg {
		info, err := portscan.GetInterfaceInfo(m.networkInterface)
		return ifaceInfoMsg{info: info, err: err}
	}
}

func (m model) currentList() []*store.DeviceRecord {
	return orderedDevices(m.store)
}
