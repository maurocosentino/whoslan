package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"whoslan/internal/i18n"
	"whoslan/internal/store"
)

// ensureSudo le pide la contraseña de sudo al usuario de forma interactiva
// ANTES de lanzar la TUI. Esto evita que sudo intente pedir la contraseña
// en background más tarde (durante un escaneo automático), lo cual
// compite por stdin con Bubble Tea y rompe los atajos de teclado.
func ensureSudo() error {
	cmd := exec.Command("sudo", "-v")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// keepSudoAlive refresca el cache de sudo cada 4 minutos en background,
// para que no expire en sesiones largas y vuelva a pedir contraseña
// mientras la TUI ya está corriendo.
func keepSudoAlive() {
	for {
		time.Sleep(4 * time.Minute)
		exec.Command("sudo", "-v").Run() // no debería pedir contraseña si el cache sigue vigente
	}
}

func main() {
	iface := flag.String("interface", "enp1s0", "Interfaz de red a escanear (ej: enp1s0, wlan0)")
	lang := flag.String("lang", "en", "Interface language (es, en)")
	flag.Parse()

	if err := ensureSudo(); err != nil {
		fmt.Println("Se necesita acceso sudo para escanear la red:", err)
		os.Exit(1)
	}
	go keepSudoAlive()

	s, err := store.Load()
	if err != nil {
		fmt.Println("Error cargando historial:", err)
		os.Exit(1)
	}

	m := model{
		store:            s,
		networkInterface: *iface,
		t:                i18n.Load(*lang),
		lang:             *lang,
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
