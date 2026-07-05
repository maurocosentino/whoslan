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
}

var es = Strings{
	Title: 			 "whoslan — dispositivos",
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