package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"whoslan/internal/portscan"
	"whoslan/internal/scanner"
)

// DeviceRecord es el estado histórico de un dispositivo: cuándo se lo
// vio por primera y última vez, y si está online en este momento.
type DeviceRecord struct {
	IP           string    `json:"ip"`
	MAC          string    `json:"mac"`
	Vendor       string    `json:"vendor"`
	Name         string    `json:"name"`
	FirstSeen    time.Time `json:"first_seen"`
	LastSeen     time.Time `json:"last_seen"`
	Online       bool      `json:"online"`
	Acknowledged bool      `json:"acknowledged"`
	MissedScans  int       `json:"missed_scans"`
}

// PortRecord es el estado histórico de un puerto en LISTEN: cuándo se
// vio por primera y última vez, y si el usuario ya lo reconoció.
type PortRecord struct {
	Port         uint32    `json:"port"`
	Protocol     string    `json:"protocol"`
	ProcessName  string    `json:"process_name"`
	FirstSeen    time.Time `json:"first_seen"`
	LastSeen     time.Time `json:"last_seen"`
	Acknowledged bool      `json:"acknowledged"`
}

const maxMissedScans = 3

// Store mantiene el historial de dispositivos, indexado por MAC
// (a diferencia de la IP, la MAC no cambia por DHCP).
type Store struct {
	path      string
	portsPath string
	Devices   map[string]*DeviceRecord
	Ports     map[string]*PortRecord 
}

// Load abre (o crea si no existe) el archivo de historial en disco.
func Load() (*Store, error) {
	path, err := storePath("history.json")
	if err != nil {
		return nil, err
	}
	portsPath, err := storePath("ports_history.json")
	if err != nil {
		return nil, err
	}

	s := &Store{
		path:      path,
		portsPath: portsPath,
		Devices:   make(map[string]*DeviceRecord),
		Ports:     make(map[string]*PortRecord),
	}

	if data, err := os.ReadFile(path); err == nil {
		json.Unmarshal(data, &s.Devices)
	}
	if data, err := os.ReadFile(portsPath); err == nil {
		json.Unmarshal(data, &s.Ports)
	}

	return s, nil
}

// Save persiste el historial actual a disco.
func (s *Store) Save() error {
	data, err := json.MarshalIndent(s.Devices, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

func (s *Store) ApplyScan(found []scanner.Device) {
	now := time.Now()
	seenNow := make(map[string]bool)

	for _, d := range found {
		seenNow[d.MAC] = true
		record, exists := s.Devices[d.MAC]

		if !exists {
			s.Devices[d.MAC] = &DeviceRecord{
				IP: d.IP, MAC: d.MAC, Vendor: d.Vendor,
				FirstSeen: now, LastSeen: now, Online: true,
				MissedScans: 0,
			}
			continue
		}

		if !record.Online {
			// Estaba offline, arranca una nueva sesión de conexión.
			record.FirstSeen = now
		}
		record.IP = d.IP
		record.LastSeen = now
		record.Online = true
		record.MissedScans = 0 // respondió, reseteamos el contador de fallos
	}

	// Los que no aparecieron ahora: sumamos un fallo, y solo marcamos
	// offline si acumularon demasiados fallos consecutivos.
	for mac, record := range s.Devices {
		if !seenNow[mac] && record.Online {
			record.MissedScans++
			if record.MissedScans >= maxMissedScans {
				record.Online = false
			}
		}
	}
}


func (s *Store) SavePorts() error {
	data, err := json.MarshalIndent(s.Ports, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.portsPath, data, 0644)
}

// ApplyPortScan actualiza el historial de puertos con los resultados de
// un escaneo nuevo, marcando como no reconocidos los que nunca se vieron.
func (s *Store) ApplyPortScan(found []portscan.ListeningPort) {
	now := time.Now()

	for _, p := range found {
		key := fmt.Sprintf("%s:%d", p.Protocol, p.Port)
		record, exists := s.Ports[key]

		if !exists {
			s.Ports[key] = &PortRecord{
				Port: p.Port, Protocol: p.Protocol, ProcessName: p.ProcessName,
				FirstSeen: now, LastSeen: now, Acknowledged: false,
			}
			continue
		}

		record.LastSeen = now
		record.ProcessName = p.ProcessName // el proceso dueño puede cambiar
	}
}

// TogglePortAcknowledge alterna el estado de reconocimiento de un puerto.
func (s *Store) TogglePortAcknowledge(protocol string, port uint32) {
	key := fmt.Sprintf("%s:%d", protocol, port)
	if record, exists := s.Ports[key]; exists {
		record.Acknowledged = !record.Acknowledged
	}
}

func storePath(filename string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".local", "share", "whoslan")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, filename), nil
}

// ToggleAcknowledge alterna el estado de reconocimiento de un dispositivo:
// si estaba reconocido, vuelve a marcarlo como pendiente de revisar, y viceversa.
func (s *Store) ToggleAcknowledge(mac string) {
	if record, exists := s.Devices[mac]; exists {
		record.Acknowledged = !record.Acknowledged
	}
}

// SetName asigna un nombre personalizado a un dispositivo por su MAC.
func (s *Store) SetName(mac, name string) {
	if record, exists := s.Devices[mac]; exists {
		record.Name = name
	}
}
