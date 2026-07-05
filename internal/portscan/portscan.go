package portscan

import (
	"fmt"

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