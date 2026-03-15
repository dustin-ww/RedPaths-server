package dgraphutil

import (
	"RedPaths-server/pkg/model/core"
	"RedPaths-server/pkg/model/core/res"
	"RedPaths-server/pkg/model/utils"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"
	"time"

	"github.com/dgraph-io/dgo/v210"
)

// --- Types -------------------------------------------------------------------

type HopConfig struct {
	Predicate  core.Predicate
	ObjectType string // only relevant for the last hop
}

type leafAssertion struct {
	data       map[string]any
	subjectUID string
}

type dedupEntry[T any] struct {
	result        *res.EntityResult[T]
	assertionUIDs map[string]struct{}
}

// --- N-Hop main function -----------------------------------------------------

func GetEntitiesWithAssertionsNHop[T any](
	ctx context.Context,
	tx *dgo.Txn,
	subjectUID string,
	hops []HopConfig,
	objectFields []string,
	queryName string,
) ([]*res.EntityResult[T], error) {
	if tx == nil {
		return nil, fmt.Errorf("transaction cannot be nil")
	}
	if len(hops) == 0 {
		return nil, fmt.Errorf("at least one hop required")
	}
	if queryName == "" {
		queryName = "getNHopEntities"
	}
	if len(objectFields) == 0 {
		objectFields = []string{"uid", "dgraph.type"}
	}

	query := buildNHopQuery(queryName, hops, objectFields)
	log.Printf("[%s] Generated query:\n%s", queryName, query)

	// Only $subjectUID is needed as a variable — predicates are inlined
	variables := map[string]string{
		"$subjectUID": subjectUID,
	}
	log.Printf("[%s] Variables: %+v", queryName, variables)

	resp, err := tx.QueryWithVars(ctx, query, variables)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	log.Printf("[%s] Raw response: %s", queryName, string(resp.Json))

	var rawResult map[string]any
	if err := json.Unmarshal(resp.Json, &rawResult); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %w", err)
	}

	subjects := extractArray(rawResult, "subject")
	if len(subjects) == 0 {
		log.Printf("[%s] No subject found with UID %s", queryName, subjectUID)
		return []*res.EntityResult[T]{}, nil
	}

	log.Printf("[%s] Found subject node, starting traversal with hops=%d", queryName, len(hops))
	leafAssertions := traverseToLeafAssertions(subjects[0], subjectUID, len(hops)-1)
	log.Printf("[%s] Total leaf assertions found: %d", queryName, len(leafAssertions))

	dedupMap := make(map[string]*dedupEntry[T])

	for _, leaf := range leafAssertions {
		objectRaw, ok := leaf.data["object"]
		if !ok {
			log.Printf("[%s] warn: leaf assertion has no 'object' field, data keys: %v", queryName, mapKeys(leaf.data))
			continue
		}

		objectJSON, err := json.Marshal(objectRaw)
		if err != nil {
			log.Printf("[%s] warn: failed to marshal object: %v", queryName, err)
			continue
		}

		log.Printf("[%s] objectJSON: %s", queryName, string(objectJSON))

		// Try array first, fall back to single object
		var entities []T
		if err := json.Unmarshal(objectJSON, &entities); err != nil {
			var single T
			if err2 := json.Unmarshal(objectJSON, &single); err2 != nil {
				log.Printf("[%s] warn: failed to unmarshal object: %v\njson: %s",
					queryName, err2, string(objectJSON))
				continue
			}
			entities = []T{single}
		}

		log.Printf("[%s] entities count after unmarshal: %d", queryName, len(entities))

		for _, entity := range entities {
			objectUID := extractUID(entity)
			ts := extractTime(leaf.data, "assertion.timestamp")
			assertionUID := extractString(leaf.data, "assertion_uid")

			// Resolve subject UID from query result, fall back to traversal value
			resolvedSubjectUID := leaf.subjectUID
			if subjectData := extractArray(leaf.data, "assertion_subject"); len(subjectData) > 0 {
				if uid := extractString(subjectData[0], "uid"); uid != "" {
					resolvedSubjectUID = uid
				}
			}

			log.Printf("[%s] processing entity: objectUID=%s assertionUID=%s subjectUID=%s predicate=%s",
				queryName, objectUID, assertionUID, resolvedSubjectUID,
				extractString(leaf.data, "assertion.predicate"))

			assertion := &core.Assertion{
				UID:                 assertionUID,
				Predicate:           core.Predicate(extractString(leaf.data, "assertion.predicate")),
				Method:              core.Method(extractString(leaf.data, "assertion.method")),
				Source:              extractString(leaf.data, "assertion.source"),
				Confidence:          extractFloat(leaf.data, "assertion.confidence"),
				Status:              core.Status(extractString(leaf.data, "assertion.status")),
				Timestamp:           ts,
				Note:                extractString(leaf.data, "assertion.note"),
				MarkedAsHighValue:   extractBool(leaf.data, "assertion.high_value_marked"),
				HasDiscoveredParent: extractBool(leaf.data, "assertion.has_discovered_parent"),
				Subject:             &utils.UIDRef{UID: resolvedSubjectUID},
				Object:              &utils.UIDRef{UID: objectUID},
			}

			if entry, found := dedupMap[objectUID]; found {
				// Entity already exists — only add assertion if not seen before
				if _, exists := entry.assertionUIDs[assertionUID]; !exists {
					entry.assertionUIDs[assertionUID] = struct{}{}
					entry.result.Assertions = append(entry.result.Assertions, assertion)
					entry.result.Metadata.AssertionCount++
					log.Printf("[%s] added assertion %s to existing entity %s", queryName, assertionUID, objectUID)
				} else {
					log.Printf("[%s] skipping duplicate assertion %s for entity %s", queryName, assertionUID, objectUID)
				}
			} else {
				// New entity
				dedupMap[objectUID] = &dedupEntry[T]{
					result: &res.EntityResult[T]{
						Entity:     entity,
						Assertions: []*core.Assertion{assertion},
						Metadata: &res.ResultMetadata{
							Source:         assertion.Source,
							ScanTimestamp:  ts,
							EntityCount:    1,
							AssertionCount: 1,
						},
					},
					assertionUIDs: map[string]struct{}{
						assertionUID: {},
					},
				}
				log.Printf("[%s] new entity added: objectUID=%s", queryName, objectUID)
			}
		}
	}

	deduped := make([]*res.EntityResult[T], 0, len(dedupMap))
	for _, entry := range dedupMap {
		deduped = append(deduped, entry.result)
	}

	log.Printf("[%s] Found %d entities after dedup (subject: %s, hops: %d)",
		queryName, len(deduped), subjectUID, len(hops))

	return deduped, nil
}

// --- Query builder -----------------------------------------------------------

func buildNHopQuery(queryName string, hops []HopConfig, objectFields []string) string {
	fieldsStr := strings.Join(objectFields, "\n\t\t\t\t\t\t")

	lastHop := hops[len(hops)-1]
	typeFilter := ""
	if strings.TrimSpace(lastHop.ObjectType) != "" {
		typeFilter = fmt.Sprintf("@filter(type(%s))", lastHop.ObjectType)
	}

	// Innermost block: last hop with all assertion fields.
	// Predicates are inlined directly — DGraph does not support query variables
	// inside nested reverse edge filters.
	inner := fmt.Sprintf(`~assertion.subject @filter(eq(assertion.predicate, "%s")) {
					assertion_uid: uid
					assertion.predicate
					assertion.method
					assertion.source
					assertion.confidence
					assertion.status
					assertion.timestamp
					assertion.note
					assertion.high_value_marked
					assertion.has_discovered_parent
					assertion_subject: assertion.subject {
						uid
					}
					object: assertion.object %s {
						%s
					}
				}`, string(lastHop.Predicate), typeFilter, fieldsStr)

	// Wrap intermediate hops from inside out
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

// --- Traversal ---------------------------------------------------------------

// traverseToLeafAssertions recursively walks through intermediate hops.
// remainingHops=0 means this node directly contains the target assertions.
func traverseToLeafAssertions(node map[string]any, nodeUID string, remainingHops int) []leafAssertion {
	log.Printf("[traverse] nodeUID=%s remainingHops=%d", nodeUID, remainingHops)

	if remainingHops == 0 {
		assertionMaps := extractAssertionMaps(node)
		log.Printf("[traverse] leaf node: found %d assertions", len(assertionMaps))
		result := make([]leafAssertion, 0, len(assertionMaps))
		for _, a := range assertionMaps {
			log.Printf("[traverse] leaf assertion predicate=%s uid=%s",
				extractString(a, "assertion.predicate"),
				extractString(a, "assertion_uid"))
			result = append(result, leafAssertion{
				data:       a,
				subjectUID: nodeUID,
			})
		}
		return result
	}

	var result []leafAssertion
	assertionArr := extractArray(node, "~assertion.subject")
	log.Printf("[traverse] ~assertion.subject count=%d", len(assertionArr))

	for _, assertion := range assertionArr {
		objects := extractArray(assertion, "assertion.object")
		log.Printf("[traverse] assertion.object count=%d", len(objects))
		for _, obj := range objects {
			uid := extractString(obj, "uid")
			log.Printf("[traverse] following to object uid=%s", uid)
			result = append(result, traverseToLeafAssertions(obj, uid, remainingHops-1)...)
		}
	}
	return result
}

func extractAssertionMaps(node map[string]any) []map[string]any {
	arr := extractArray(node, "~assertion.subject")
	result := make([]map[string]any, 0, len(arr))
	result = append(result, arr...)
	return result
}

// --- 1-hop convenience wrapper -----------------------------------------------

func GetEntitiesWithAssertions[T any](
	ctx context.Context,
	tx *dgo.Txn,
	subjectUID string,
	predicate core.Predicate,
	objectType string,
	objectFields []string,
	queryName string,
) ([]*res.EntityResult[T], error) {
	return GetEntitiesWithAssertionsNHop[T](
		ctx,
		tx,
		subjectUID,
		[]HopConfig{
			{Predicate: predicate, ObjectType: objectType},
		},
		objectFields,
		queryName,
	)
}

// --- Helper functions --------------------------------------------------------

func extractArray(m map[string]any, key string) []map[string]any {
	val, ok := m[key]
	if !ok {
		return nil
	}
	arr, ok := val.([]any)
	if !ok {
		// Single object — wrap in slice
		if mp, ok := val.(map[string]any); ok {
			return []map[string]any{mp}
		}
		return nil
	}
	result := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		if mp, ok := item.(map[string]any); ok {
			result = append(result, mp)
		}
	}
	return result
}

func extractString(m map[string]any, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func extractFloat(m map[string]any, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

func extractBool(m map[string]any, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func extractTime(m map[string]any, key string) time.Time {
	s := extractString(m, key)
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		t, _ = time.Parse(time.RFC3339, s)
	}
	return t
}

func extractUID[T any](entity T) string {
	ev := reflect.ValueOf(entity)
	if ev.Kind() == reflect.Ptr {
		if ev.IsNil() {
			return ""
		}
		ev = ev.Elem()
	}
	if ev.Kind() == reflect.Struct {
		if f := ev.FieldByName("UID"); f.IsValid() && f.Kind() == reflect.String {
			return f.String()
		}
	}
	return ""
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
