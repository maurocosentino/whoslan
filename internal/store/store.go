package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

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

const maxMissedScans = 3

// Store mantiene el historial de dispositivos, indexado por MAC
// (a diferencia de la IP, la MAC no cambia por DHCP).
type Store struct {
	path    string
	Devices map[string]*DeviceRecord
}

// Load abre (o crea si no existe) el archivo de historial en disco.
func Load() (*Store, error) {
	path, err := storePath()
	if err != nil {
		return nil, err
	}

	s := &Store{
		path:    path,
		Devices: make(map[string]*DeviceRecord),
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return s, nil // primera vez que corre, historial vacío
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, &s.Devices); err != nil {
		return nil, err
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

func storePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".local", "share", "whoslan")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "history.json"), nil
}

// Acknowledge marca un dispositivo como reconocido, sacándole la alerta
// de "nuevo" para siempre (hasta que se le haga reset manual, si hiciera falta).
func (s *Store) Acknowledge(mac string) {
	if record, exists := s.Devices[mac]; exists {
		record.Acknowledged = true
	}
}

// SetName asigna un nombre personalizado a un dispositivo por su MAC.
func (s *Store) SetName(mac, name string) {
	if record, exists := s.Devices[mac]; exists {
		record.Name = name
	}
}
