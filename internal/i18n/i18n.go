package i18n

// Strings agrupa todos los textos visibles de la interfaz para un idioma.
type Strings struct {
	TitleOnline     string
	TitleHistory    string
	ColName         string
	ColIP           string
	ColMAC          string
	ColStatus       string
	ColDuration     string
	ColAlert 		string
	StatusOnline    string
	StatusOffline   string
	ConnectedFor    string
	DisconnectedFor string
	HelpBar         string
	RenamePrompt    string
	RenameHelp      string
	ScanError       string
}

var es = Strings{
	TitleOnline:     "whoslan — dispositivos online",
	TitleHistory:    "whoslan — historial completo",
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
	HelpBar:         "(↑/↓ para moverte · h para historial/online · a para reconocer · r para renombrar · q para salir · escaneo cada %s)",
	RenamePrompt:    "Nuevo nombre: ",
	RenameHelp:      "(enter para confirmar · esc para cancelar)",
	ScanError:       "Error escaneando: %v",
}

var en = Strings{
	TitleOnline:     "whoslan — online devices",
	TitleHistory:    "whoslan — full history",
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
	HelpBar:         "(↑/↓ to move · h for history/online · a to acknowledge · r to rename · q to quit · scanning every %s)",
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