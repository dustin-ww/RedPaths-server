package dgraph

import (
	"RedPaths-server/pkg/model/core"
	"RedPaths-server/pkg/model/core/res"
	"context"
	"fmt"
	"reflect"
	"strings"

	"encoding/json"

	"github.com/dgraph-io/dgo/v210"
)

// -----------------------------------------------------------------------------
// Types
// -----------------------------------------------------------------------------

type ExistenceSource string

const (
	ExistenceSourceHierarchy ExistenceSource = "hierarchy"
	ExistenceSourceProject   ExistenceSource = "project"
	ExistenceSourceNotFound  ExistenceSource = "not_found"
)

type UniqueFilterMode string

const (
	FilterModeAND UniqueFilterMode = "AND"
	FilterModeOR  UniqueFilterMode = "OR"
)

type UniqueFieldFilter struct {
	Field string
	Value string
}

type ExistenceResult[T any] struct {
	Found    bool
	Entities []*res.EntityResult[T]
	FoundVia ExistenceSource
}

type scoredCandidate[T any] struct {
	Result *res.EntityResult[T]
	Score  float64
}

// -----------------------------------------------------------------------------
// Public API
// -----------------------------------------------------------------------------

func CheckEntityExists[T any](
	ctx context.Context,
	tx *dgo.Txn,
	projectUID string,
	objectType string,
	filters []UniqueFieldFilter,
	filterMode UniqueFilterMode,
	objectFields []string,
	hierarchyHops []HopConfig,
) (*ExistenceResult[T], error) {
	if tx == nil {
		return nil, fmt.Errorf("transaction cannot be nil")
	}

	// Früh raus wenn keine sinnvollen Filter vorhanden
	hasValue := false
	for _, f := range filters {
		if strings.TrimSpace(f.Value) != "" {
			hasValue = true
			break
		}
	}
	if !hasValue {
		return &ExistenceResult[T]{Found: false, FoundVia: ExistenceSourceNotFound}, nil
	}

	// Phase 1: search via full hierarchy
	if len(hierarchyHops) > 0 {
		result, err := searchViaHierarchy[T](
			ctx, tx, projectUID, objectType,
			filters, filterMode, objectFields, hierarchyHops,
		)
		if err != nil {
			return nil, err
		}
		if result.Found {
			return result, nil
		}
	}

	// Phase 2: search directly at project level (orphaned / unplaced)
	return searchAtProjectLevel[T](
		ctx, tx, projectUID, objectType,
		filters, filterMode, objectFields,
	)
}

func ScoreCandidates[T any](
	candidates []*res.EntityResult[T],
	filters []UniqueFieldFilter,
	minScore float64,
) []scoredCandidate[T] {
	scored := make([]scoredCandidate[T], 0, len(candidates))

	for _, c := range candidates {
		matched := 0
		total := 0
		for _, f := range filters {
			if f.Value == "" {
				continue
			}
			total++
			if reflectFieldMatches(c.Entity, f.Field, f.Value) {
				matched++
			}
		}
		if total == 0 {
			continue
		}
		score := float64(matched) / float64(total)
		if score >= minScore {
			scored = append(scored, scoredCandidate[T]{Result: c, Score: score})
		}
	}

	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].Score > scored[i].Score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	return scored
}

func BestCandidate[T any](
	candidates []*res.EntityResult[T],
	filters []UniqueFieldFilter,
	minScore float64,
) *scoredCandidate[T] {
	scored := ScoreCandidates(candidates, filters, minScore)
	if len(scored) == 0 {
		return nil
	}
	return &scored[0]
}

// -----------------------------------------------------------------------------
// Internal: hierarchy search
// -----------------------------------------------------------------------------

func searchViaHierarchy[T any](
	ctx context.Context,
	tx *dgo.Txn,
	projectUID string,
	objectType string,
	filters []UniqueFieldFilter,
	filterMode UniqueFilterMode,
	objectFields []string,
	hops []HopConfig,
) (*ExistenceResult[T], error) {
	query := buildExistenceNHopQuery(
		"existenceCheckHierarchy",
		hops, objectFields,
		objectType, filters, filterMode,
	)

	resp, err := tx.QueryWithVars(ctx, query, map[string]string{
		"$subjectUID": projectUID,
	})
	if err != nil {
		return nil, fmt.Errorf("hierarchy existence query failed: %w", err)
	}

	entities, err := parseExistenceResponse[T](resp.Json, "existenceCheckHierarchy")
	if err != nil {
		return nil, err
	}
	if len(entities) == 0 {
		return &ExistenceResult[T]{Found: false, FoundVia: ExistenceSourceNotFound}, nil
	}

	return &ExistenceResult[T]{
		Found:    true,
		Entities: entities,
		FoundVia: ExistenceSourceHierarchy,
	}, nil
}

// -----------------------------------------------------------------------------
// Internal: project-level (orphaned) search
// -----------------------------------------------------------------------------

func searchAtProjectLevel[T any](
	ctx context.Context,
	tx *dgo.Txn,
	projectUID string,
	objectType string,
	filters []UniqueFieldFilter,
	filterMode UniqueFilterMode,
	objectFields []string,
) (*ExistenceResult[T], error) {
	fieldsStr := strings.Join(objectFields, "\n\t\t\t\t\t")
	combinedFilter := buildCombinedFilter(objectType, filters, filterMode)

	query := fmt.Sprintf(`query existenceCheckProject($subjectUID: string) {
		subject(func: uid($subjectUID)) {
			~assertion.subject {
				object: assertion.object %s {
					%s
				}
			}
		}
	}`, combinedFilter, fieldsStr)

	resp, err := tx.QueryWithVars(ctx, query, map[string]string{
		"$subjectUID": projectUID,
	})
	if err != nil {
		return nil, fmt.Errorf("project-level existence query failed: %w", err)
	}

	entities, err := parseExistenceResponse[T](resp.Json, "existenceCheckProject")
	if err != nil {
		return nil, err
	}
	if len(entities) == 0 {
		return &ExistenceResult[T]{Found: false, FoundVia: ExistenceSourceNotFound}, nil
	}

	return &ExistenceResult[T]{
		Found:    true,
		Entities: entities,
		FoundVia: ExistenceSourceProject,
	}, nil
}

// -----------------------------------------------------------------------------
// Query builder
// -----------------------------------------------------------------------------

func buildExistenceNHopQuery(
	queryName string,
	hops []HopConfig,
	objectFields []string,
	objectType string,
	filters []UniqueFieldFilter,
	filterMode UniqueFilterMode,
) string {
	fieldsStr := strings.Join(objectFields, "\n\t\t\t\t\t\t")
	combinedFilter := buildCombinedFilter(objectType, filters, filterMode)

	lastHop := hops[len(hops)-1]
	inner := fmt.Sprintf(`~assertion.subject @filter(eq(assertion.predicate, "%s")) {
				object: assertion.object %s {
					%s
				}
			}`, string(lastHop.Predicate), combinedFilter, fieldsStr)

	for i := len(hops) - 2; i >= 0; i-- {
		inner = fmt.Sprintf(`~assertion.subject @filter(eq(assertion.predicate, "%s")) {
			assertion.object {
				uid
				%s
			}
		}`, string(hops[i].Predicate), inner)
	}

	return fmt.Sprintf(`query %s($subjectUID: string) {
		subject(func: uid($subjectUID)) {
			%s
		}
	}`, queryName, inner)
}

// buildCombinedFilter kombiniert type() und Feldfilter zu einem einzigen @filter.
// Dgraph erlaubt nur eine @filter-Direktive pro Ausdruck — mehrere hintereinander
// sind ein Query-Fehler ("Use AND, OR and round brackets instead of multiple
// filter directives").
//
// Ergebnis-Beispiele:
//
//	type only:        @filter(type(Host))
//	field only:       @filter(eq(host.ip, "10.0.0.1"))
//	type + 1 field:   @filter(type(Host) AND eq(host.ip, "10.0.0.1"))
//	type + 2 fields:  @filter(type(Host) AND (eq(host.ip, "10.0.0.1") OR eq(host.hostname, "srv")))
func buildCombinedFilter(objectType string, filters []UniqueFieldFilter, mode UniqueFilterMode) string {
	// Feldfilter sammeln — leere Werte überspringen
	fieldParts := make([]string, 0, len(filters))
	for _, f := range filters {
		if strings.TrimSpace(f.Value) == "" {
			continue
		}
		fieldParts = append(fieldParts, fmt.Sprintf(`eq(%s, "%s")`, f.Field, f.Value))
	}

	hasType := strings.TrimSpace(objectType) != ""
	hasFields := len(fieldParts) > 0

	switch {
	case !hasType && !hasFields:
		return ""

	case hasType && !hasFields:
		return fmt.Sprintf("@filter(type(%s))", objectType)

	case !hasType && hasFields:
		if len(fieldParts) == 1 {
			return fmt.Sprintf("@filter(%s)", fieldParts[0])
		}
		return fmt.Sprintf("@filter(%s)", strings.Join(fieldParts, " "+string(mode)+" "))

	default: // hasType && hasFields
		var fieldExpr string
		if len(fieldParts) == 1 {
			fieldExpr = fieldParts[0]
		} else {
			// Mehrere Feldfilter in Klammern — mode (AND/OR) gilt nur zwischen Feldern
			fieldExpr = fmt.Sprintf("(%s)", strings.Join(fieldParts, " "+string(mode)+" "))
		}
		return fmt.Sprintf("@filter(type(%s) AND %s)", objectType, fieldExpr)
	}
}

// -----------------------------------------------------------------------------
// Response parsing
// -----------------------------------------------------------------------------

func parseExistenceResponse[T any](rawJSON []byte, queryName string) ([]*res.EntityResult[T], error) {
	var rawResult map[string]any
	if err := json.Unmarshal(rawJSON, &rawResult); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %w", err)
	}

	subjects := extractArray(rawResult, "subject")
	if len(subjects) == 0 {
		return nil, nil
	}

	var objectNodes []map[string]any
	collectObjectNodes(subjects[0], &objectNodes)

	if len(objectNodes) == 0 {
		return nil, nil
	}

	results := make([]*res.EntityResult[T], 0, len(objectNodes))
	for _, node := range objectNodes {
		nodeJSON, err := json.Marshal(node)
		if err != nil {
			continue
		}
		var entity T
		if err := json.Unmarshal(nodeJSON, &entity); err != nil {
			continue
		}
		results = append(results, &res.EntityResult[T]{
			Entity:     entity,
			Assertions: []*core.Assertion{},
			Metadata:   nil,
		})
	}

	return results, nil
}

// collectObjectNodes extracts "object" arrays from the response tree.
func collectObjectNodes(node map[string]any, out *[]map[string]any) {
	if objects, ok := node["object"]; ok {
		switch v := objects.(type) {
		case []any:
			for _, item := range v {
				if m, ok := item.(map[string]any); ok {
					*out = append(*out, m)
				}
			}
		case map[string]any:
			*out = append(*out, v)
		}
		return
	}

	assertionArr := extractArray(node, "~assertion.subject")
	for _, a := range assertionArr {
		objArr := extractArray(a, "assertion.object")
		for _, obj := range objArr {
			collectObjectNodes(obj, out)
		}
		collectObjectNodes(a, out)
	}
}

// -----------------------------------------------------------------------------
// Reflection helper
// -----------------------------------------------------------------------------

func reflectFieldMatches[T any](entity T, field, value string) bool {
	ev := reflect.ValueOf(entity)
	if ev.Kind() == reflect.Ptr {
		if ev.IsNil() {
			return false
		}
		ev = ev.Elem()
	}
	if ev.Kind() != reflect.Struct {
		return false
	}

	t := ev.Type()
	fieldLower := strings.ToLower(field)

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		jsonTag := strings.Split(f.Tag.Get("json"), ",")[0]

		if jsonTag == field || strings.ToLower(f.Name) == fieldLower || jsonTag == fieldLower {
			return fmt.Sprintf("%v", ev.Field(i).Interface()) == value
		}
	}
	return false
}
