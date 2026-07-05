package main

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"whoslan/internal/i18n"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if wsMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = wsMsg.Width
		m.height = wsMsg.Height
		return m, nil
	}

	if m.renaming {
		return m.updateRenaming(msg)
	}

	switch m.screen {
	case screenMenu:
		return m.updateMenu(msg)
	case screenDevices:
		return m.updateDevices(msg)
	case screenPorts:
		return m.updatePorts(msg)
	case screenConnections:
		return m.updateConnections(msg)
	case screenInterface:
		return m.updateInterface(msg)
	}
	return m, nil
}

func (m model) updateMenu(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch keyMsg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.menuCursor > 0 {
			m.menuCursor--
		}
		return m, nil
	case "down", "j":
		if m.menuCursor < len(m.t.MenuItems)-1 {
			m.menuCursor++
		}
		return m, nil
	case "enter":
		return m.selectMenuItem(m.t.MenuItems[m.menuCursor].Key)
	}

	// Atajo directo: si la tecla coincide con el Key de algún item
	// del menú, lo seleccionamos sin necesidad de navegar primero.
	for i, item := range m.t.MenuItems {
		if item.Key == keyMsg.String() {
			m.menuCursor = i
			return m.selectMenuItem(item.Key)
		}
	}

	return m, nil
}

// selectMenuItem centraliza qué pasa al confirmar una opción del menú,
// sea por Enter (con el cursor ya posicionado) o por atajo directo de letra.
func (m model) selectMenuItem(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "d":
		m.screen = screenDevices
		m.cursor = 0
		return m, m.doScan()
	case "p":
		m.screen = screenPorts
		m.portsCursor = 0
		return m, m.doPortScan()
	case "c":
		m.screen = screenConnections
		m.connCursor = 0
		return m, m.doConnScan()
	case "i":
		m.screen = screenInterface
		return m, m.doGetInterfaceInfo()
	case "l":
		if m.lang == "es" {
			m.lang = "en"
		} else {
			m.lang = "es"
		}
		m.t = i18n.Load(m.lang)
		return m, nil
	case "q":
		return m, tea.Quit
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

func (m model) updatePorts(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case portScanResultMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.ports = msg.ports
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.screen = screenMenu
			return m, nil
		case "up", "k":
			if m.portsCursor > 0 {
				m.portsCursor--
			}
		case "down", "j":
			if m.portsCursor < len(m.ports)-1 {
				m.portsCursor++
			}
		case "s":
			return m, m.doPortScan()
		}
	}
	return m, nil
}

func (m model) updateConnections(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case connScanResultMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.connections = msg.connections
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.screen = screenMenu
			return m, nil
		case "up", "k":
			if m.connCursor > 0 {
				m.connCursor--
			}
		case "down", "j":
			if m.connCursor < len(m.connections)-1 {
				m.connCursor++
			}
		case "s":
			return m, m.doConnScan()
		}
	}
	return m, nil
}

func (m model) updateInterface(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ifaceInfoMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.ifaceInfo = msg.info
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.screen = screenMenu
			return m, nil
		case "s":
			return m, m.doGetInterfaceInfo()
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
