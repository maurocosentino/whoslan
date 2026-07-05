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
	AppTitle        string
	AppSubtitle     string
	MenuItems       []MenuItem
	MenuHelp        string
	//Ports
	ColPort     string
	ColProtocol string
	ColProcess  string
	ColPID      string
	PortsTitle  string
	PortsHelp   []HelpItem
	PortsError  string

	ColLocal      string
	ColRemote     string
	ColConnStatus string
	ConnTitle     string
	ConnHelp      []HelpItem
	ConnError     string

	InterfaceTitle string
	LabelInterface string
	LabelLocalIP   string
	LabelNetmask   string
	LabelGateway   string
	LabelPublicIP  string
	InterfaceHelp  []HelpItem
	InterfaceError string
	NotAvailable   string
}

var es = Strings{
	Title:       "whoslan — dispositivos",
	AppTitle:    "whoslan",
	AppSubtitle: "Panel de red y seguridad",
	MenuItems: []MenuItem{
		{"d", "Dispositivos", "Ver y gestionar dispositivos conectados"},
		{"p", "Puertos", "Ver puertos abiertos y procesos"},
		{"c", "Conexiones", "Ver conexiones de red activas"},
		{"i", "Interfaz", "Ver IP, gateway y datos de red"},
		{"l", "Idioma", "Cambiar entre español e inglés"},
		{"q", "Salir", "Cerrar whoslan"},
	},
	MenuHelp:        "(↑/↓ para moverte · enter para seleccionar)",
	ColName:         "Nombre",
	ColIP:           "IP",
	ColMAC:          "MAC",
	ColStatus:       "Estado",
	ColDuration:     "Duración",
	ColAlert:        "",
	StatusOnline:    "Online",
	StatusOffline:   "Offline",
	ConnectedFor:    "hace %s",
	DisconnectedFor: "%s",
	HelpItems: []HelpItem{
		{"↑/↓", "moverte"},
		{"s", "escanear"},
		{"a", "reconocer"},
		{"r", "renombrar"},
		{"esc", "volver"},
	},
	RenamePrompt: "Nuevo nombre: ",
	RenameHelp:   "(enter para confirmar · esc para cancelar)",
	ScanError:    "Error escaneando: %v",
	//ports
	ColPort:     "Puerto",
	ColProtocol: "Protocolo",
	ColProcess:  "Proceso",
	ColPID:      "PID",
	PortsTitle:  "whoslan — puertos abiertos",
	PortsHelp: []HelpItem{
		{"↑/↓", "moverte"},
		{"s", "escanear"},
		{"a", "reconocer"},
		{"esc", "volver"},
	},
	PortsError:    "Error escaneando puertos: %v",
	ColLocal:      "Local",
	ColRemote:     "Remoto",
	ColConnStatus: "Estado",
	ConnTitle:     "whoslan — conexiones activas",
	ConnHelp: []HelpItem{
		{"↑/↓", "moverte"},
		{"s", "escanear"},
		{"esc", "volver"},
	},
	ConnError:      "Error escaneando conexiones: %v",
	InterfaceTitle: "whoslan — interfaz de red",
	LabelInterface: "Interfaz",
	LabelLocalIP:   "IP local",
	LabelNetmask:   "Máscara",
	LabelGateway:   "Gateway",
	LabelPublicIP:  "IP pública",
	InterfaceHelp: []HelpItem{
		{"s", "actualizar"},
		{"esc", "volver"},
	},
	InterfaceError: "Error obteniendo datos de red: %v",
	NotAvailable:   "no disponible",
}

var en = Strings{
	Title:       "whoslan — devices",
	AppTitle:    "whoslan",
	AppSubtitle: "Network & security panel",
	MenuItems: []MenuItem{
		{"d", "Devices", "View and manage connected devices"},
		{"p", "Ports", "View open ports and processes"},
		{"c", "Connections", "View active network connections"},
		{"i", "Interface", "View IP, gateway and network info"},
		{"l", "Language", "Switch between English and Spanish"},
		{"q", "Quit", "Exit whoslan"},
	},
	MenuHelp:        "(↑/↓ to move · enter to select)",
	ColName:         "Name",
	ColIP:           "IP",
	ColMAC:          "MAC",
	ColStatus:       "Status",
	ColDuration:     "Duration",
	ColAlert:        "",
	StatusOnline:    "Online",
	StatusOffline:   "Offline",
	ConnectedFor:    "%s ago",
	DisconnectedFor: "%s",
	HelpItems: []HelpItem{
		{"↑/↓", "move"},
		{"s", "scan"},
		{"a", "acknowledge"},
		{"r", "rename"},
		{"esc", "back"},
	},
	RenamePrompt: "New name: ",
	RenameHelp:   "(enter to confirm · esc to cancel)",
	ScanError:    "Scan error: %v",
	//ports
	ColPort:     "Port",
	ColProtocol: "Protocol",
	ColProcess:  "Process",
	ColPID:      "PID",
	PortsTitle:  "whoslan — open ports",
	PortsHelp: []HelpItem{
		{"↑/↓", "move"},
		{"s", "scan"},
		{"a", "acknowledge"},
		{"esc", "back"},
	},
	PortsError:    "Error scanning ports: %v",
	ColLocal:      "Local",
	ColRemote:     "Remote",
	ColConnStatus: "Status",
	ConnTitle:     "whoslan — active connections",
	ConnHelp: []HelpItem{
		{"↑/↓", "move"},
		{"s", "scan"},
		{"esc", "back"},
	},
	ConnError:      "Error scanning connections: %v",
	InterfaceTitle: "whoslan — network interface",
	LabelInterface: "Interface",
	LabelLocalIP:   "Local IP",
	LabelNetmask:   "Netmask",
	LabelGateway:   "Gateway",
	LabelPublicIP:  "Public IP",
	InterfaceHelp: []HelpItem{
		{"s", "refresh"},
		{"esc", "back"},
	},
	InterfaceError: "Error getting network info: %v",
	NotAvailable:   "not available",
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
