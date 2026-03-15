package core

import (
	"RedPaths-server/pkg/model/utils"
	"time"
)

type Predicate string
type Method string
type Status string

// ----------------------
// Predicates
// ----------------------
const (
	PredicatePartOfProject      Predicate = "part_of_project" // z.B. AD gehört zu Projekt
	PredicateLinkedDomain       Predicate = "linked_domain"   // z.B. Domain gehört zu AD
	PredicateHasHost            Predicate = "has_host"        // z.B. Host gehört zu Domain/Project
	PredicateHasActiveDirectory Predicate = "has_ad"
	PredicateHasDomain          Predicate = "has_domain"
	PredicateDetectedByScan     Predicate = "detected_by_scan" // z.B. via Scan erkannt
	PredicateCompromised        Predicate = "compromised"      // z.B. Sicherheitsereignis
	PredicateContains           Predicate = "contains"
	PredicateLocates            Predicate = "locates"
	PredicateHasGPOLink         Predicate = "has_gpo_link"
	PredicateParent             Predicate = "parent"
	PredicateRuns               Predicate = "runs"
	PredicateLinksTo            Predicate = "links_to"
)

// ----------------------
// Methods
// ----------------------
const (
	MethodDirectAdd    Method = "direct_add"    // Manuelle Erstellung
	MethodScanDetected Method = "scan_detected" // Scan-basiert
	MethodImported     Method = "imported"      // Importiert aus externen Quellen
	MethodInference    Method = "inferred"      // Vom System abgeleitet
)

// ----------------------
// Status
// ----------------------
const (
	StatusValidated   Status = "direct_add"
	StatusTentative   Status = "scan_detected"
	StatusInvalidated Status = "imported"
	StatusExpired     Status = "inferred"
)

type Assertion struct {
	UID                 string    `json:"uid,omitempty"`
	DType               []string  `json:"dgraph.type,omitempty"`
	Predicate           Predicate `json:"assertion.predicate"`
	Method              Method    `json:"assertion.method"`
	Source              string    `json:"assertion.source"`
	Confidence          float64   `json:"assertion.confidence"`
	Status              Status    `json:"assertion.status"`
	Timestamp           time.Time `json:"assertion.timestamp"`
	Note                string    `json:"assertion.node"`
	MarkedAsHighValue   bool      `json:"assertion.high_value_marked"`
	HasDiscoveredParent bool      `json:"assertion.has_discovered_parent"`

	Subject *utils.UIDRef `json:"assertion.subject"`
	Object  *utils.UIDRef `json:"assertion.object"`
}

func New(subjectUID, objectUID, source string, pred Predicate, method Method, confidence float64) *Assertion {
	return &Assertion{
		Subject:    &utils.UIDRef{UID: subjectUID},
		Object:     &utils.UIDRef{UID: objectUID},
		Predicate:  pred,
		Method:     method,
		Source:     source,
		Confidence: confidence,
		Status:     StatusValidated,
		Timestamp:  time.Now(),
	}
}
