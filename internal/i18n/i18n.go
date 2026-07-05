package i18n

// Strings agrupa todos los textos visibles de la interfaz para un idioma.
type Strings struct {
	Title           string
	ColName         string
	ColIP           string
	ColMAC          string
	ColStatus       string
	ColDuration     string
	ColAlert        string
	StatusOnline    string
	StatusOffline   string
	ConnectedFor    string
	DisconnectedFor string
	HelpItems       []HelpItem
	RenamePrompt    string
	RenameHelp      string
	ScanError       string
	AppTitle    	string
	AppSubtitle 	string
	MenuItems   	[]MenuItem
	MenuHelp 		string
}

var es = Strings{
	Title: 			 "whoslan — dispositivos",
	AppTitle:    "whoslan",
	AppSubtitle: "Panel de red y seguridad",
	MenuItems: []MenuItem{
		{"d", "Dispositivos", "Ver y gestionar dispositivos conectados"},
		{"q", "Salir", "Cerrar whoslan"},
	},
	MenuHelp: 		 "(↑/↓ para moverte · enter para seleccionar)",
	ColName:         "Nombre",
	ColIP:           "IP",
	ColMAC:          "MAC",
	ColStatus:       "Estado",
	ColDuration:     "Duración",
	ColAlert: 		 "",
	StatusOnline:    "Online",
	StatusOffline:   "Offline",
	ConnectedFor:    "hace %s",
	DisconnectedFor: "%s",
	HelpItems: []HelpItem{
		{"↑/↓", "moverte"},
		{"s", "escanear"},
		{"a", "reconocer"},
		{"r", "renombrar"},
		{"q", "salir"},
	},
	RenamePrompt:    "Nuevo nombre: ",
	RenameHelp:      "(enter para confirmar · esc para cancelar)",
	ScanError:       "Error escaneando: %v",
}

var en = Strings{
	Title: 			 "whoslan — devices",
	AppTitle:    "whoslan",
	AppSubtitle: "Network & security panel",
	MenuItems: []MenuItem{
		{"d", "Devices", "View and manage connected devices"},
		{"q", "Quit", "Exit whoslan"},
	},
	MenuHelp: 		 "(↑/↓ to move · enter to select)",
	ColName:         "Name",
	ColIP:           "IP",
	ColMAC:          "MAC",
	ColStatus:       "Status",
	ColDuration:     "Duration",
	ColAlert: 		 "",
	StatusOnline:    "Online",
	StatusOffline:   "Offline",
	ConnectedFor:    "%s ago",
	DisconnectedFor: "%s",
	HelpItems: []HelpItem{
		{"↑/↓", "move"},
		{"s", "scan"},
		{"a", "acknowledge"},
		{"r", "rename"},
		{"q", "quit"},
	},
	RenamePrompt:    "New name: ",
	RenameHelp:      "(enter to confirm · esc to cancel)",
	ScanError:       "Scan error: %v",
}

// MenuItem representa una opción del menú principal.
type MenuItem struct {
	Key         string
	Label       string
	Description string
}

// Load devuelve el conjunto de strings para el código de idioma dado.
// Si no se reconoce, cae a español por defecto.
func Load(lang string) Strings {
	switch lang {
	case "en":
		return en
	default:
		return es
	}
}

// HelpItem representa un atajo de teclado y su descripción.
type HelpItem struct {
	Key    string
	Action string
}