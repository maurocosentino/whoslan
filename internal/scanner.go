package scanner

import (
	"os/exec"
	"regexp"
	"strings"
)

// Device representa un dispositivo detectado en la red en un momento dado.
// Por ahora solo tiene los datos crudos del escaneo; el historial
// (FirstSeen/LastSeen/Online) lo va a manejar el paquete store.
type Device struct {
	IP     string
	MAC    string
	Vendor string
}

// Scan ejecuta arp-scan sobre la interfaz indicada y devuelve los
// dispositivos encontrados.
func Scan(iface string) ([]Device, error) {
	cmd := exec.Command("sudo", "arp-scan", "--interface="+iface, "--localnet")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}
	return parseArpScan(string(output)), nil
}

func parseArpScan(output string) []Device {
	var devices []Device
	lineRegex := regexp.MustCompile(`^(\d+\.\d+\.\d+\.\d+)\s+([0-9a-fA-F:]{17})\s+(.+)$`)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		matches := lineRegex.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		devices = append(devices, Device{
			IP:     matches[1],
			MAC:    matches[2],
			Vendor: strings.TrimSpace(matches[3]),
		})
	}
	return devices
}