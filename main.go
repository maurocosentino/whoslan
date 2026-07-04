package main

import (
	"fmt"
	"os/exec"
)

func main() {
	cmd := exec.Command("sudo", "arp-scan", "--interface=enp1s0", "--localnet")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Println("Error ejecutando arp-scan:", err)
		fmt.Println(string(output))
		return
	}

	fmt.Println(string(output))
}