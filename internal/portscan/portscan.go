package portscan

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	psnet "github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// ListeningPort representa un puerto en estado LISTEN en la máquina local.
type ListeningPort struct {
	Port        uint32
	Protocol    string // "tcp" o "udp"
	ProcessName string
	PID         int32
}

// Connection representa una conexión de red activa (no solo LISTEN).
type Connection struct {
	Protocol    string
	LocalAddr   string
	RemoteAddr  string
	Status      string
	ProcessName string
	PID         int32
}

// InterfaceInfo agrupa los datos de configuración de red local.
type InterfaceInfo struct {
	Name     string
	LocalIP  string
	Netmask  string
	Gateway  string
	PublicIP string
}

// GetInterfaceInfo arma un resumen de la configuración de red usando la
// interfaz indicada. La IP pública requiere conexión a internet; si falla,
// queda vacía en vez de cortar el resto de la información.
func GetInterfaceInfo(iface string) (InterfaceInfo, error) {
	info := InterfaceInfo{Name: iface}

	ifaceObj, err := netInterfaceByName(iface)
	if err != nil {
		return info, err
	}

	addrs, err := ifaceObj.Addrs()
	if err != nil {
		return info, err
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.To4() == nil {
			continue
		}
		info.LocalIP = ipNet.IP.String()
		ones, _ := ipNet.Mask.Size()
		info.Netmask = fmt.Sprintf("/%d", ones)
		break
	}

	info.Gateway = getDefaultGateway()
	info.PublicIP = getPublicIP()

	return info, nil
}

func netInterfaceByName(name string) (*net.Interface, error) {
	return net.InterfaceByName(name)
}

// getDefaultGateway lee el gateway por defecto desde /proc/net/route
// (específico de Linux, evita depender de parsear "ip route" como texto).
func getDefaultGateway() string {
	data, err := os.ReadFile("/proc/net/route")
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		if fields[1] != "00000000" { // destino 0.0.0.0 = ruta por defecto
			continue
		}
		gwHex := fields[2]
		gwBytes := make([]byte, 4)
		fmt.Sscanf(gwHex[6:8], "%02x", &gwBytes[0])
		fmt.Sscanf(gwHex[4:6], "%02x", &gwBytes[1])
		fmt.Sscanf(gwHex[2:4], "%02x", &gwBytes[2])
		fmt.Sscanf(gwHex[0:2], "%02x", &gwBytes[3])
		return net.IP(gwBytes).String()
	}
	return ""
}

// getPublicIP consulta un servicio externo simple para obtener la IP
// pública. Si no hay conexión, devuelve cadena vacía sin fallar.
func getPublicIP() string {
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("https://api.ipify.org")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(body))
}

// ScanConnections devuelve las conexiones activas que no están en modo
// LISTEN (es decir, conexiones reales hacia/desde otra IP:puerto).
func ScanConnections() ([]Connection, error) {
	conns, err := psnet.Connections("inet")
	if err != nil {
		return nil, err
	}

	var result []Connection
	for _, c := range conns {
		if c.Status == "LISTEN" || c.Status == "NONE" {
			continue
		}
		if c.Raddr.IP == "" {
			continue // sin dirección remota, no es una conexión real
		}

		name := "?"
		if c.Pid > 0 {
			if p, err := process.NewProcess(c.Pid); err == nil {
				if n, err := p.Name(); err == nil {
					name = n
				}
			}
		}

		protocol := "tcp"
		if c.Type == 2 {
			protocol = "udp"
		}

		result = append(result, Connection{
			Protocol:    protocol,
			LocalAddr:   fmt.Sprintf("%s:%d", c.Laddr.IP, c.Laddr.Port),
			RemoteAddr:  fmt.Sprintf("%s:%d", c.Raddr.IP, c.Raddr.Port),
			Status:      c.Status,
			ProcessName: name,
			PID:         c.Pid,
		})
	}

	return result, nil
}

// Scan devuelve todos los puertos en LISTEN, con el proceso dueño si se
// puede resolver (puede requerir permisos para algunos procesos ajenos).
func Scan() ([]ListeningPort, error) {
	conns, err := psnet.Connections("inet")
	if err != nil {
		return nil, err
	}

	var ports []ListeningPort
	seen := make(map[string]bool) // evita duplicados (mismo puerto, mismo PID)

	for _, c := range conns {
		if c.Status != "LISTEN" {
			continue
		}

		key := fmt.Sprintf("%d-%d", c.Laddr.Port, c.Pid)
		if seen[key] {
			continue
		}
		seen[key] = true

		name := "?"
		if c.Pid > 0 {
			if p, err := process.NewProcess(c.Pid); err == nil {
				if n, err := p.Name(); err == nil {
					name = n
				}
			}
		}

		protocol := "tcp"
		if c.Type == 2 { // syscall.SOCK_DGRAM
			protocol = "udp"
		}

		ports = append(ports, ListeningPort{
			Port:        c.Laddr.Port,
			Protocol:    protocol,
			ProcessName: name,
			PID:         c.Pid,
		})
	}

	return ports, nil
}
