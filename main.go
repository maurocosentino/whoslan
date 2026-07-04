package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

type Device struct {
	IP       string
	MAC      string
	Vendor   string
}

func main() {
	cmd := exec.Command("sudo", "arp-scan", "--interface=enp1s0", "--localnet")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error ejecutando arp-scan:", err)
		fmt.Println(string(output))
		return
	}

	devices := parseArpScan(string(output))

	for _, d := range devices {
		fmt.Printf("%-16s %-18s %s\n", d.IP, d.MAC, d.Vendor)
	}
}

func parseArpScan(output string) []Device {
	var devices []Device

	// Cada línea de dispositivo tiene el formato:
	// IP<tab>MAC<tab>Vendor
	lineRegex := regexp.MustCompile(`^(\d+\.\d+\.\d+\.\d+)\s+([0-9a-fA-F:]{17})\s+(.+)$`)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		matches := lineRegex.FindStringSubmatch(line)
		if matches == nil {
			continue // no es una línea de dispositivo (headers, footers, etc.)
		}

		devices = append(devices, Device{
			IP:     matches[1],
			MAC:    matches[2],
			Vendor: strings.TrimSpace(matches[3]),
		})
	}

	return devices
}