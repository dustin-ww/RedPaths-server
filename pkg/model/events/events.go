package events

type SystemEventType int

const (
	ModuleStarted SystemEventType = iota
)

/*type EventType int

const (
	NewDomainDiscovered EventType = iota
	NewHostDiscovered
	NewServiceDiscovered
	NewUserDiscovered

	DomainExistenceConfirmed
	HostExistenceConfirmed
	ServiceExistenceConfirmed

	DomainStatusUnknown
	HostStatusUnknown
	ServiceStatusUnknown
)

func (s EventType) String() string {
	/*switch s {
	case StatusNew:
		return "New"
	case StatusInProgress:
		return "InProgress"
	case StatusCompleted:
		return "Completed"
	case StatusArchived:
		return "Archived"
	default:
		return "Unknown"
	}*/

// EventType repräsentiert einen SSE Event-Typ
type EventType string

// Event-Typen als Konstanten
const (
	ScanStart    EventType = "scan_start"
	ScanProgress EventType = "scan_progress"
	ScanComplete EventType = "scan_complete"
	ScanError    EventType = "scan_error"

	ModuleStart    EventType = "module_start"
	ModuleComplete EventType = "module_complete"
	ModuleError    EventType = "module_error"

	DomainDiscovered EventType = "domain_discovered"

	HostDiscovered  EventType = "host_discovered"
	PortFound       EventType = "port_found"
	ServiceDetected EventType = "service_detected"

	VulnFound    EventType = "vulnerability_found"
	VulnAnalyzed EventType = "vulnerability_analyzed"
)

// String gibt den String-Wert des EventType zurück
func (e EventType) String() string {
	return string(e)
}

// IsValid prüft, ob der EventType gültig ist
func (e EventType) IsValid() bool {
	switch e {
	case ScanStart, ScanProgress, ScanComplete, ScanError,
		ModuleStart, ModuleComplete, ModuleError,
		HostDiscovered, PortFound, ServiceDetected, DomainDiscovered,
		VulnFound, VulnAnalyzed:
		return true
	}
	return false
}
