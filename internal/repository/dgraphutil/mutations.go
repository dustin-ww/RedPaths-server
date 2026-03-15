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
	"strconv"
	"strings"
	"time"

	"github.com/dgraph-io/dgo/v210"
	"github.com/dgraph-io/dgo/v210/protos/api"
	"github.com/google/uuid"
)

func MapToStruct(m map[string]interface{}, out interface{}) error {
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

// FindEntityViaAssertionAndRelation findet eine Entity über eine zweistufige Traversierung:
// 1. Subject -> Assertion -> IntermediateEntity
// 2. IntermediateEntity -> directRelation -> TargetEntity (mit Feldfilter)
func FindEntityViaAssertionAndRelation[T any](
	ctx context.Context,
	tx *dgo.Txn,
	subjectUID string,
	assertionPredicate core.Predicate,
	intermediateType string,
	directRelation string,
	targetType string,
	filterField string,
	filterValue interface{},
	targetFields []string,
) (*T, error) {
	if tx == nil {
		return nil, fmt.Errorf("transaction cannot be nil")
	}

	dgType, dgValue, err := getDgraphTypeAndValue(filterValue)
	if err != nil {
		return nil, fmt.Errorf("type handling error: %w", err)
	}

	// Default fields wenn nicht angegeben
	if len(targetFields) == 0 {
		targetFields = []string{"uid", "dgraph.type"}
	}
	fieldsStr := strings.Join(targetFields, "\n                            ")

	query := fmt.Sprintf(`
		query FindViaAssertionAndRelation($subjectUID: string, $predicate: string, $filterValue: %s) {
			subject(func: uid($subjectUID)) {
				assertions: ~assertion.subject @filter(eq(assertion.predicate, $predicate)) {
					intermediate: assertion.object @filter(type(%s)) {
						uid
						target: %s @filter(type(%s) AND eq(%s, $filterValue)) {
							%s
						}
					}
				}
			}
		}
	`, dgType, intermediateType, directRelation, targetType, filterField, fieldsStr)

	variables := map[string]string{
		"$subjectUID":  subjectUID,
		"$predicate":   string(assertionPredicate),
		"$filterValue": dgValue,
	}

	resp, err := tx.QueryWithVars(ctx, query, variables)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	var rawResult struct {
		Subject []struct {
			Assertions []struct {
				Intermediate []struct {
					UID    string          `json:"uid"`
					Target json.RawMessage `json:"target"`
				} `json:"intermediate"`
			} `json:"assertions"`
		} `json:"subject"`
	}

	if err := json.Unmarshal(resp.Json, &rawResult); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %w", err)
	}

	if len(rawResult.Subject) == 0 {
		log.Printf("[FindViaAssertionAndRelation] No subject found with UID %s", subjectUID)
		return nil, nil
	}

	// Durchsuche alle Assertions und Intermediate Nodes
	for _, assertion := range rawResult.Subject[0].Assertions {
		for _, intermediate := range assertion.Intermediate {
			if len(intermediate.Target) == 0 {
				continue
			}

			// Versuche als Array zu parsen
			var targetArray []T
			if err := json.Unmarshal(intermediate.Target, &targetArray); err == nil {
				if len(targetArray) > 0 {
					return &targetArray[0], nil
				}
				continue
			}

			// Fallback: Als einzelnes Objekt
			var singleTarget T
			if err := json.Unmarshal(intermediate.Target, &singleTarget); err == nil {
				return &singleTarget, nil
			}
		}
	}

	return nil, nil
}

// ExistsEntityViaAssertionAndRelation prüft ob eine Entity über die zweistufige Traversierung existiert
func ExistsEntityViaAssertionAndRelation(
	ctx context.Context,
	tx *dgo.Txn,
	subjectUID string,
	assertionPredicate core.Predicate,
	intermediateType string,
	directRelation string,
	targetType string,
	filterField string,
	filterValue interface{},
) (bool, string, error) {
	if tx == nil {
		return false, "", fmt.Errorf("transaction cannot be nil")
	}

	dgType, dgValue, err := getDgraphTypeAndValue(filterValue)
	if err != nil {
		return false, "", fmt.Errorf("type handling error: %w", err)
	}

	query := fmt.Sprintf(`
		query ExistsViaAssertionAndRelation($subjectUID: string, $predicate: string, $filterValue: %s) {
			subject(func: uid($subjectUID)) {
				assertions: ~assertion.subject @filter(eq(assertion.predicate, $predicate)) {
					intermediate: assertion.object @filter(type(%s)) {
						target: %s @filter(type(%s) AND eq(%s, $filterValue)) {
							uid
						}
					}
				}
			}
		}
	`, dgType, intermediateType, directRelation, targetType, filterField)

	variables := map[string]string{
		"$subjectUID":  subjectUID,
		"$predicate":   string(assertionPredicate),
		"$filterValue": dgValue,
	}

	resp, err := tx.QueryWithVars(ctx, query, variables)
	if err != nil {
		return false, "", fmt.Errorf("query failed: %w", err)
	}

	var result struct {
		Subject []struct {
			Assertions []struct {
				Intermediate []struct {
					Target []struct {
						UID string `json:"uid"`
					} `json:"target"`
				} `json:"intermediate"`
			} `json:"assertions"`
		} `json:"subject"`
	}

	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return false, "", fmt.Errorf("unmarshal failed: %w", err)
	}

	if len(result.Subject) == 0 {
		return false, "", nil
	}

	for _, assertion := range result.Subject[0].Assertions {
		for _, intermediate := range assertion.Intermediate {
			if len(intermediate.Target) > 0 {
				return true, intermediate.Target[0].UID, nil
			}
		}
	}

	return false, "", nil
}

func UpdateAndGet[T any](
	ctx context.Context,
	tx *dgo.Txn,
	uid string,
	actor string,
	fields map[string]interface{},
	get func(context.Context, *dgo.Txn, string) (*T, error),
) (*T, error) {

	fields["uid"] = uid
	fields["last_seen_by"] = actor
	fields["last_seen_at"] = time.Now().Format(time.RFC3339)

	jsonData, err := json.Marshal(fields)
	if err != nil {
		return nil, err
	}

	if _, err := tx.Mutate(ctx, &api.Mutation{SetJson: jsonData}); err != nil {
		return nil, err
	}

	return get(ctx, tx, uid)
}

// GetEntitiesByRelation retrieves entities that have a relationship to a specific UID
func GetEntitiesByRelation[T any](
	ctx context.Context,
	tx *dgo.Txn,
	entityType string,
	relationName string,
	targetUID string,
	fields []string,
) ([]T, error) {
	if tx == nil {
		return nil, fmt.Errorf("transaction cannot be nil")
	}

	fieldsStr := strings.Join(fields, "\n")

	query := fmt.Sprintf(`
		query GetEntitiesByRelation($targetUID: string) {
			%s(func: has(%s)) @filter(uid_in(%s, $targetUID)) {
				%s
			}
		}
	`, entityType, relationName, relationName, fieldsStr)

	vars := map[string]string{"$targetUID": targetUID}
	resp, err := tx.QueryWithVars(ctx, query, vars)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}

	var rawResult map[string]json.RawMessage
	if err := json.Unmarshal(resp.Json, &rawResult); err != nil {
		return nil, fmt.Errorf("unmarshal raw result error: %w", err)
	}

	entitiesData, ok := rawResult[entityType]
	if !ok {
		return []T{}, nil
	}

	var entities []T
	if err := json.Unmarshal(entitiesData, &entities); err != nil {
		return nil, fmt.Errorf("unmarshal entities error: %w", err)
	}

	return entities, nil
}

// AddRelation creates a relationship between two nodes
func AddRelation(ctx context.Context, tx *dgo.Txn, sourceUID, targetUID, relationName string) error {
	if tx == nil {
		return fmt.Errorf("transaction cannot be nil")
	}

	log.Printf("[DEBUG-REL] AddRelation: sourceUID=%q targetUID=%q rel=%q",
		sourceUID, targetUID, relationName)

	if sourceUID == "" {
		log.Printf("[ERROR-REL] sourceUID is EMPTY → Dgraph will create a new node")
	}
	if targetUID == "" {
		log.Printf("[ERROR-REL] targetUID is EMPTY → Dgraph will create a new node")
	}

	if !strings.HasPrefix(sourceUID, "0x") {
		log.Printf("[ERROR-REL] sourceUID invalid → %q", sourceUID)
	}
	if !strings.HasPrefix(targetUID, "0x") {
		log.Printf("[ERROR-REL] targetUID invalid → %q", targetUID)
	}

	update := map[string]interface{}{
		"uid": sourceUID,
		relationName: []map[string]string{
			{"uid": targetUID},
		},
	}

	return executeMutation(ctx, tx, update)
}

func InitCreateMetadata(meta *core.RedPathsMetadata, actor string) {
	now := time.Now().UTC()

	meta.CreatedAt = now
	meta.DiscoveredAt = now
	meta.LastSeenAt = now
	meta.DiscoveredBy = actor
	meta.LastSeenBy = actor
}

// OldCreateEntity creates a new entity with a unique blank node ID and returns the assigned UID
func OldCreateEntity(ctx context.Context, tx *dgo.Txn, dtype string, entity interface{}) (string, error) {
	if tx == nil {
		return "", fmt.Errorf("transaction cannot be nil")
	}

	jsonBytes, err := json.Marshal(entity)
	if err != nil {
		return "", fmt.Errorf("marshal entity error: %w", err)
	}

	var entityMap map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &entityMap); err != nil {
		return "", fmt.Errorf("unmarshal to map error: %w", err)
	}

	// Generate unique blank node ID
	blankID := uuid.New().String()
	entityMap["uid"] = fmt.Sprintf("_:%s", blankID)
	entityMap["dgraph.type"] = dtype

	jsonData, err := json.Marshal(entityMap)
	if err != nil {
		return "", fmt.Errorf("marshal error: %w", err)
	}

	mu := &api.Mutation{SetJson: jsonData}
	assigned, err := tx.Mutate(ctx, mu)
	if err != nil {
		return "", fmt.Errorf("mutation error: %w", err)
	}

	return assigned.Uids[blankID], nil
}

func CreateEntity[T any](ctx context.Context, tx *dgo.Txn, dtype string, entity T) (T, error) {
	var zero T

	if tx == nil {
		return zero, fmt.Errorf("transaction cannot be nil")
	}

	jsonBytes, err := json.Marshal(entity)
	if err != nil {
		return zero, fmt.Errorf("marshal entity error: %w", err)
	}

	var entityMap map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &entityMap); err != nil {
		return zero, fmt.Errorf("unmarshal to map error: %w", err)
	}

	blankID := uuid.New().String()
	entityMap["uid"] = fmt.Sprintf("_:%s", blankID)
	entityMap["dgraph.type"] = dtype

	jsonData, err := json.Marshal(entityMap)
	if err != nil {
		return zero, fmt.Errorf("marshal error: %w", err)
	}

	mu := &api.Mutation{SetJson: jsonData}
	assigned, err := tx.Mutate(ctx, mu)
	if err != nil {
		return zero, fmt.Errorf("mutation error: %w", err)
	}

	if assignedUID, ok := assigned.Uids[blankID]; ok {
		if reflect.ValueOf(entity).Kind() == reflect.Ptr {
			reflect.ValueOf(entity).Elem().FieldByName("UID").SetString(assignedUID)
		}
		return entity, nil
	}

	return zero, fmt.Errorf("no UID assigned from mutation")
}

func ExistsByFieldInDomain(
	ctx context.Context,
	tx *dgo.Txn,
	domainUID string,
	entityType string,
	fieldName string,
	fieldValue interface{},
) (bool, error) {
	if tx == nil {
		return false, fmt.Errorf("transaction cannot be nil")
	}

	dgType, dgValue, err := getDgraphTypeAndValue(fieldValue)
	if err != nil {
		return false, fmt.Errorf("type handling error: %w", err)
	}

	var entityPath string
	switch entityType {
	case "Host":
		entityPath = "has_host"
	case "User":
		entityPath = "has_user"
	case "Service":
		entityPath = "has_host { has_service }"
	default:
		return false, fmt.Errorf("unsupported entity type for domain: %s", entityType)
	}

	query := fmt.Sprintf(`
		query ExistsByFieldInDomain($fieldValue: %s, $domainUID: string) {
			domain(func: uid($domainUID)) @filter(type(Domain)) {
				%s @filter(type(%s) AND eq(%s, $fieldValue)) {
					uid
				}
			}
		}
	`, dgType, entityPath, entityType, fieldName)

	vars := map[string]string{
		"$fieldValue": dgValue,
		"$domainUID":  domainUID,
	}

	res, err := tx.QueryWithVars(ctx, query, vars)
	if err != nil {
		return false, fmt.Errorf("query error: %w", err)
	}

	var result map[string][]interface{}
	if err := json.Unmarshal(res.Json, &result); err != nil {
		return false, fmt.Errorf("unmarshal error: %w", err)
	}

	domains, ok := result["domain"]
	if !ok || len(domains) == 0 {
		return false, nil
	}

	return hasEntitiesInPath(domains[0], entityType), nil
}

func ExistsByFieldInProject(
	ctx context.Context,
	tx *dgo.Txn,
	projectID string,
	entityType string,
	fieldName string,
	fieldValue interface{},
) (bool, error) {

	if tx == nil {
		return false, fmt.Errorf("transaction cannot be nil")
	}

	dgType, dgValue, err := getDgraphTypeAndValue(fieldValue)
	if err != nil {
		return false, fmt.Errorf("type handling error: %w", err)
	}

	var directPath, domainPath string

	switch entityType {
	case "Domain":
		directPath = "has_domain"

	case "Host":
		directPath = "has_host"
		domainPath = "has_domain { has_host }"

	case "User":
		directPath = "has_user"
		domainPath = "has_domain { has_user }"

	case "Service":
		domainPath = "has_domain { has_host { has_service } }"

	case "RedPathsModule":
		directPath = "has_redpaths_modules"

	case "Target":
		directPath = "has_target"

	default:
		return false, fmt.Errorf("unknown entity type: %s", entityType)
	}

	query := fmt.Sprintf(`
		query ExistsByField($fieldValue: %s, $projectID: string) {
			project(func: uid($projectID)) @filter(type(Project)) {

				%s

				%s
			}
		}
	`,
		dgType,
		buildFilteredPath("direct", directPath, entityType, fieldName),
		buildFilteredPath("via_domain", domainPath, entityType, fieldName),
	)

	vars := map[string]string{
		"$fieldValue": dgValue,
		"$projectID":  projectID,
	}

	res, err := tx.QueryWithVars(ctx, query, vars)
	if err != nil {
		return false, fmt.Errorf("query error: %w", err)
	}

	var result struct {
		Project []struct {
			Direct    []interface{} `json:"direct"`
			ViaDomain []interface{} `json:"via_domain"`
		} `json:"project"`
	}

	if err := json.Unmarshal(res.Json, &result); err != nil {
		return false, fmt.Errorf("unmarshal error: %w", err)
	}

	if len(result.Project) == 0 {
		return false, nil
	}

	return len(result.Project[0].Direct) > 0 || len(result.Project[0].ViaDomain) > 0, nil
}

func buildFilteredPath(alias, path, entityType, fieldName string) string {
	if path == "" {
		return ""
	}

	if !strings.Contains(path, "{") {
		return fmt.Sprintf(`
			%s: %s @filter(type(%s) AND eq(%s, $fieldValue)) {
				uid
			}
		`, alias, path, entityType, fieldName)
	}

	return fmt.Sprintf(`
		%s: %s {
			@filter(type(%s) AND eq(%s, $fieldValue)) {
				uid
			}
		}
	`, alias, path, entityType, fieldName)
}

// NodeExistsWithField checks if a node exists with the given field-value pair
func ExistsByField(ctx context.Context, tx *dgo.Txn, entityType string, fieldName string, fieldValue interface{}) (bool, error) {
	if tx == nil {
		return false, fmt.Errorf("transaction cannot be nil")
	}

	var query string
	var vars map[string]string

	switch v := fieldValue.(type) {
	case string:
		query = fmt.Sprintf(`
            query ExistsByField($fieldValue: string) {
                entity(func: type(%s)) @filter(eq(%s, $fieldValue)) {
                    uid
                }
            }
        `, entityType, fieldName)
		vars = map[string]string{"$fieldValue": v}

	case int, int32, int64:
		query = fmt.Sprintf(`
            query ExistsByField($fieldValue: int) {
                entity(func: type(%s)) @filter(eq(%s, $fieldValue)) {
                    uid
                }
            }
        `, entityType, fieldName)
		vars = map[string]string{"$fieldValue": fmt.Sprintf("%d", v)}

	case float32, float64:
		query = fmt.Sprintf(`
            query ExistsByField($fieldValue: float) {
                entity(func: type(%s)) @filter(eq(%s, $fieldValue)) {
                    uid
                }
            }
        `, entityType, fieldName)
		vars = map[string]string{"$fieldValue": fmt.Sprintf("%g", v)}

	case bool:
		query = fmt.Sprintf(`
            query ExistsByField($fieldValue: bool) {
                entity(func: type(%s)) @filter(eq(%s, $fieldValue)) {
                    uid
                }
            }
        `, entityType, fieldName)
		vars = map[string]string{"$fieldValue": fmt.Sprintf("%t", v)}

	default:
		// uid oder unbekannter Typ: direkt interpolieren, keine Variable
		query = fmt.Sprintf(`
            {
                entity(func: type(%s)) @filter(eq(%s, %v)) {
                    uid
                }
            }
        `, entityType, fieldName, fieldValue)
		vars = nil
	}

	var res *api.Response
	var err error
	if vars != nil {
		res, err = tx.QueryWithVars(ctx, query, vars)
	} else {
		res, err = tx.Query(ctx, query)
	}
	if err != nil {
		return false, fmt.Errorf("query error: %w", err)
	}

	var result struct {
		Entity []struct {
			UID string `json:"uid"`
		} `json:"entity"`
	}
	if err := json.Unmarshal(res.Json, &result); err != nil {
		return false, fmt.Errorf("unmarshal error: %w", err)
	}

	return len(result.Entity) > 0, nil
}

// HasAttribute checks if a node has a specific attribute
func HasAttribute(ctx context.Context, tx *dgo.Txn, uid, attributeName string) (bool, error) {
	if tx == nil {
		return false, fmt.Errorf("transaction cannot be nil")
	}

	query := fmt.Sprintf(`
		query HasAttribute($uid: string) {
			node(func: uid($uid)) {
				%s
			}
		}
	`, attributeName)

	vars := map[string]string{"$uid": uid}
	res, err := tx.QueryWithVars(ctx, query, vars)
	if err != nil {
		return false, fmt.Errorf("query error: %w", err)
	}

	var result struct {
		Node []map[string]interface{} `json:"node"`
	}

	if err := json.Unmarshal(res.Json, &result); err != nil {
		return false, fmt.Errorf("unmarshal error: %w", err)
	}

	if len(result.Node) == 0 {
		return false, nil // Node not found
	}

	_, exists := result.Node[0][attributeName]
	return exists, nil
}

// DeleteEntity deletes a node and all its attributes
func DeleteEntity(ctx context.Context, tx *dgo.Txn, uid string) error {
	if tx == nil {
		return fmt.Errorf("transaction cannot be nil")
	}

	deleteJSON := fmt.Sprintf(`{"uid": "%s"}`, uid)
	mu := &api.Mutation{DeleteJson: []byte(deleteJSON)}
	_, err := tx.Mutate(ctx, mu)
	return err
}

// DeleteEntityWithRelations deletes a node and its specified relationships
func DeleteEntityWithRelations(ctx context.Context, tx *dgo.Txn, uid string, relationNames []string) error {
	if tx == nil {
		return fmt.Errorf("transaction cannot be nil")
	}

	if len(relationNames) == 0 {
		return DeleteEntity(ctx, tx, uid)
	}

	query := buildRelationsQuery(uid, relationNames)
	resp, err := tx.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("query error: %w", err)
	}

	result, err := parseRelationQueryResult(resp.GetJson(), relationNames)
	if err != nil {
		return err
	}

	for _, relName := range relationNames {
		if entities, exists := result[relName]; exists {
			for _, entity := range entities {
				if relUID, ok := entity["uid"].(string); ok {
					if err := DeleteEntity(ctx, tx, relUID); err != nil {
						return err
					}
				}
			}
		}
	}

	return DeleteEntity(ctx, tx, uid)
}

// Helper functions
func executeMutation(ctx context.Context, tx *dgo.Txn, data map[string]interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}

	mu := &api.Mutation{SetJson: jsonData}
	_, err = tx.Mutate(ctx, mu)
	return err
}

func getDgraphTypeAndValue(fieldValue interface{}) (string, string, error) {
	switch v := fieldValue.(type) {
	case string:
		return "string", v, nil
	case int, int8, int16, int32, int64:
		return "int", fmt.Sprintf("%d", reflect.ValueOf(v).Int()), nil
	case uint, uint8, uint16, uint32, uint64:
		return "int", fmt.Sprintf("%d", reflect.ValueOf(v).Uint()), nil
	case float32, float64:
		return "float", strconv.FormatFloat(reflect.ValueOf(v).Float(), 'f', -1, 64), nil
	case bool:
		return "bool", strconv.FormatBool(v), nil
	case time.Time:
		return "datetime", v.Format(time.RFC3339), nil
	default:
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return "", "", fmt.Errorf("unsupported type %T: %w", v, err)
		}
		return "string", string(jsonBytes), nil
	}
}

func buildRelationsQuery(uid string, relations []string) string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf(`{ entity(func: uid(%s)) {`, uid))

	for _, rel := range relations {
		builder.WriteString(fmt.Sprintf(`%s { uid } `, rel))
	}

	builder.WriteString("} }")
	return builder.String()
}

func parseRelationQueryResult(jsonData []byte, relations []string) (map[string][]map[string]interface{}, error) {
	var result struct {
		Entity []map[string]interface{} `json:"entity"`
	}

	if err := json.Unmarshal(jsonData, &result); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	if len(result.Entity) == 0 {
		return nil, nil
	}

	relationMap := make(map[string][]map[string]interface{})
	for _, rel := range relations {
		if entities, ok := result.Entity[0][rel].([]interface{}); ok {
			for _, e := range entities {
				if entity, ok := e.(map[string]interface{}); ok {
					relationMap[rel] = append(relationMap[rel], entity)
				}
			}
		}
	}

	return relationMap, nil
}

func GetEntityByUID[T any](ctx context.Context, tx *dgo.Txn, uid string, queryName string, query string) (*T, error) {
	vars := map[string]string{"$uid": uid}
	resp, err := tx.QueryWithVars(ctx, query, vars)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}

	var rawResult map[string]json.RawMessage
	if err := json.Unmarshal(resp.Json, &rawResult); err != nil {
		return nil, fmt.Errorf("unmarshal raw result error: %w", err)
	}

	entitiesData, ok := rawResult[queryName]
	if !ok {
		return nil, fmt.Errorf("result does not contain field %q", queryName)
	}

	var entities []T
	if err := json.Unmarshal(entitiesData, &entities); err != nil {
		return nil, fmt.Errorf("unmarshal entities error: %w", err)
	}

	if len(entities) == 0 {
		return nil, fmt.Errorf("entity not found: %s", uid)
	}

	return &entities[0], nil
}

func GetAllEntities[T any](
	ctx context.Context,
	tx *dgo.Txn,
	entityType string,
	fields []string,
	limit int,
	offset int,
) ([]T, error) {
	fieldsStr := strings.Join(fields, "\n")

	var queryBuilder strings.Builder
	queryBuilder.WriteString(fmt.Sprintf(`
        query Get%s {
            %s(func: type(%s)`, entityType, entityType, entityType))

	if limit > 0 {
		queryBuilder.WriteString(fmt.Sprintf(`, first: %d`, limit))
	}
	if offset > 0 {
		queryBuilder.WriteString(fmt.Sprintf(`, offset: %d`, offset))
	}

	queryBuilder.WriteString(fmt.Sprintf(`) {
                %s
            }
        }`, fieldsStr))

	query := queryBuilder.String()

	resp, err := tx.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}

	var rawResult map[string]json.RawMessage
	if err := json.Unmarshal(resp.Json, &rawResult); err != nil {
		return nil, fmt.Errorf("unmarshal raw result error: %w", err)
	}

	entitiesData, ok := rawResult[entityType]
	if !ok {

		return []T{}, nil
	}

	var entities []T
	if err := json.Unmarshal(entitiesData, &entities); err != nil {
		return nil, fmt.Errorf("unmarshal entities error: %w", err)
	}

	return entities, nil
}

func GetEntityByField[T any](
	ctx context.Context,
	tx *dgo.Txn,
	entityType string,
	fieldName string,
	fieldValue interface{},
	fields []string,
) ([]T, error) {
	if tx == nil {
		return nil, fmt.Errorf("transaction cannot be nil")
	}

	fieldsStr := strings.Join(fields, "\n")

	var resp *api.Response
	var err error

	switch v := fieldValue.(type) {
	case string:
		query := fmt.Sprintf(`
            query GetEntityByField($fieldValue: string) {
                %s(func: type(%s)) @filter(eq(%s, $fieldValue)) {
                    %s
                }
            }
        `, entityType, entityType, fieldName, fieldsStr)
		resp, err = tx.QueryWithVars(ctx, query, map[string]string{"$fieldValue": v})

	case int, int32, int64:
		query := fmt.Sprintf(`
            query GetEntityByField($fieldValue: int) {
                %s(func: type(%s)) @filter(eq(%s, $fieldValue)) {
                    %s
                }
            }
        `, entityType, entityType, fieldName, fieldsStr)
		resp, err = tx.QueryWithVars(ctx, query, map[string]string{"$fieldValue": fmt.Sprintf("%d", v)})

	case float32, float64:
		query := fmt.Sprintf(`
            query GetEntityByField($fieldValue: float) {
                %s(func: type(%s)) @filter(eq(%s, $fieldValue)) {
                    %s
                }
            }
        `, entityType, entityType, fieldName, fieldsStr)
		resp, err = tx.QueryWithVars(ctx, query, map[string]string{"$fieldValue": fmt.Sprintf("%g", v)})

	case bool:
		query := fmt.Sprintf(`
            query GetEntityByField($fieldValue: bool) {
                %s(func: type(%s)) @filter(eq(%s, $fieldValue)) {
                    %s
                }
            }
        `, entityType, entityType, fieldName, fieldsStr)
		resp, err = tx.QueryWithVars(ctx, query, map[string]string{"$fieldValue": fmt.Sprintf("%t", v)})

	case uint64:
		// uid direkt interpolieren – keine Query-Variable möglich
		query := fmt.Sprintf(`
            {
                %s(func: uid(%d)) {
                    %s
                }
            }
        `, entityType, v, fieldsStr)
		resp, err = tx.Query(ctx, query)

	default:
		return nil, fmt.Errorf("unsupported field value type: %T", fieldValue)
	}

	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}

	var rawResult map[string]json.RawMessage
	if err := json.Unmarshal(resp.Json, &rawResult); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	entitiesData, ok := rawResult[entityType]
	if !ok {
		return []T{}, nil
	}

	var entities []T
	if err := json.Unmarshal(entitiesData, &entities); err != nil {
		return nil, fmt.Errorf("unmarshal entities error: %w", err)
	}

	return entities, nil
}

// DeleteEntityCascadeByTypeMap deletes a start node and recursively deletes related nodes
// according to the provided map: map[dgraphType] = []relationNamesToFollowForThatType.
// It collects UIDs (post-order) and then issues a single DeleteJson mutation.
func DeleteEntityCascadeByTypeMap(ctx context.Context, tx *dgo.Txn, startUID string, typeRelations map[string][]string) error {
	if tx == nil {
		return fmt.Errorf("transaction cannot be nil")
	}
	// helper: build the union of all relation names
	allRelsSet := map[string]struct{}{}
	for _, rels := range typeRelations {
		for _, r := range rels {
			allRelsSet[r] = struct{}{}
		}
	}
	allRels := make([]string, 0, len(allRelsSet))
	for r := range allRelsSet {
		allRels = append(allRels, r)
	}

	// cache results of node queries to avoid double-querying same uid
	type nodeData struct {
		Types   []string
		RelUids map[string][]string // rel -> []uid
	}
	cache := map[string]*nodeData{}
	visited := map[string]struct{}{}
	deleteOrder := make([]string, 0)

	// helper: build a query that asks for dgraph.type and given relations (each with uid)
	buildQuery := func(uid string, rels []string) string {
		sb := strings.Builder{}
		sb.WriteString(fmt.Sprintf(`query Node($uid: string) { node(func: uid($uid)) { uid dgraph.type `))
		for _, r := range rels {
			sb.WriteString(fmt.Sprintf("%s { uid } ", r))
		}
		sb.WriteString("} }")
		return sb.String()
	}

	// helper: query single node for union relations and parse results
	queryNode := func(uid string) (*nodeData, error) {
		if nd, ok := cache[uid]; ok {
			return nd, nil
		}
		if len(allRels) == 0 {
			// minimal query if no relations configured at all
			q := `query Node($uid: string) { node(func: uid($uid)) { uid dgraph.type } }`
			vars := map[string]string{"$uid": uid}
			res, err := tx.QueryWithVars(ctx, q, vars)
			if err != nil {
				return nil, fmt.Errorf("query error: %w", err)
			}
			var parsed struct {
				Node []map[string]interface{} `json:"node"`
			}
			if err := json.Unmarshal(res.Json, &parsed); err != nil {
				return nil, fmt.Errorf("unmarshal node error: %w", err)
			}
			nd := &nodeData{RelUids: map[string][]string{}}
			if len(parsed.Node) > 0 {
				if tRaw, ok := parsed.Node[0]["dgraph.type"]; ok {
					switch tv := tRaw.(type) {
					case []interface{}:
						for _, e := range tv {
							if s, ok := e.(string); ok {
								nd.Types = append(nd.Types, s)
							}
						}
					case string:
						nd.Types = append(nd.Types, tv)
					}
				}
			}
			cache[uid] = nd
			return nd, nil
		}
		q := buildQuery(uid, allRels)
		vars := map[string]string{"$uid": uid}
		res, err := tx.QueryWithVars(ctx, q, vars)
		if err != nil {
			return nil, fmt.Errorf("query error: %w", err)
		}
		var parsed struct {
			Node []map[string]interface{} `json:"node"`
		}
		if err := json.Unmarshal(res.Json, &parsed); err != nil {
			return nil, fmt.Errorf("unmarshal node error: %w", err)
		}
		if len(parsed.Node) == 0 {
			nd := &nodeData{RelUids: map[string][]string{}}
			cache[uid] = nd
			return nd, nil
		}
		entry := parsed.Node[0]
		nd := &nodeData{RelUids: map[string][]string{}}
		// parse types
		if tRaw, ok := entry["dgraph.type"]; ok {
			switch tv := tRaw.(type) {
			case []interface{}:
				for _, e := range tv {
					if s, ok := e.(string); ok {
						nd.Types = append(nd.Types, s)
					}
				}
			case string:
				nd.Types = append(nd.Types, tv)
			}
		}
		// parse relations (allRels) into uid lists
		for _, r := range allRels {
			if relRaw, ok := entry[r]; ok {
				if arr, ok := relRaw.([]interface{}); ok {
					for _, it := range arr {
						if m, ok := it.(map[string]interface{}); ok {
							if u, ok := m["uid"].(string); ok {
								nd.RelUids[r] = append(nd.RelUids[r], u)
							}
						}
					}
				}
			}
		}
		cache[uid] = nd
		return nd, nil
	}

	// recursive DFS post-order
	var dfs func(string) error
	dfs = func(uid string) error {
		if _, ok := visited[uid]; ok {
			return nil
		}
		visited[uid] = struct{}{}

		nd, err := queryNode(uid)
		if err != nil {
			return err
		}

		// DEBUG: Print was gefunden wurde
		fmt.Printf("DEBUG: Processing UID=%s, Types=%v\n", uid, nd.Types)
		for rel, uids := range nd.RelUids {
			if len(uids) > 0 {
				fmt.Printf("  -> Relation '%s' has %d children: %v\n", rel, len(uids), uids)
			}
		}
		// determine which relations to traverse for this node's types
		relsToTraverseSet := map[string]struct{}{}
		for _, t := range nd.Types {
			if rels, ok := typeRelations[t]; ok {
				for _, r := range rels {
					relsToTraverseSet[r] = struct{}{}
				}
			}
		}
		// if node has no dgraph.type or nothing configured for its type,
		// fallback: do not traverse any relations
		relsToTraverse := make([]string, 0, len(relsToTraverseSet))
		for r := range relsToTraverseSet {
			relsToTraverse = append(relsToTraverse, r)
		}

		// traverse children
		for _, r := range relsToTraverse {
			if children, ok := nd.RelUids[r]; ok {
				for _, childUID := range children {
					if err := dfs(childUID); err != nil {
						return err
					}
				}
			}
		}

		// post-order append (children before parent)
		deleteOrder = append(deleteOrder, uid)
		return nil
	}

	if err := dfs(startUID); err != nil {
		return err
	}

	// build delete array [{"uid":"0x1"},{"uid":"0x2"},...]
	var nquads strings.Builder
	seenForDelete := map[string]struct{}{}
	for _, u := range deleteOrder {
		if _, ok := seenForDelete[u]; ok {
			continue
		}
		seenForDelete[u] = struct{}{}
		// <uid> * * . löscht alle Triples mit diesem Subject
		nquads.WriteString(fmt.Sprintf("<%s> * * .\n", u))
	}

	mu := &api.Mutation{
		DelNquads: []byte(nquads.String()),
	}

	assigned, err := tx.Mutate(ctx, mu)

	if err != nil {
		return fmt.Errorf("delete mutation error: %w", err)
	}

	fmt.Printf("DEBUG: Mutation successful, assigned: %+v\n", assigned)

	return nil
}

// FindEntityByFieldViaAssertion findet eine Entity über eine Assertion-Beziehung
func FindEntityByFieldViaAssertion[T any](
	ctx context.Context,
	tx *dgo.Txn,
	subjectUID string,
	predicate core.Predicate,
	objectType string,
	fieldName string,
	fieldValue interface{},
	objectFields []string,
) (*T, error) {

	if tx == nil {
		return nil, fmt.Errorf("transaction cannot be nil")
	}

	dgType, dgValue, err := getDgraphTypeAndValue(fieldValue)
	if err != nil {
		return nil, fmt.Errorf("type handling error: %w", err)
	}

	fieldsStr := strings.Join(objectFields, "\n                ")

	query := fmt.Sprintf(`
        query findEntityViaAssertion($subjectUID: string, $predicate: string, $fieldValue: %s) {
            subject(func: uid($subjectUID)) {
                assertions: ~assertion.subject @filter(eq(assertion.predicate, $predicate)) {
                    uid
                    object: assertion.object @filter(type(%s) AND eq(%s, $fieldValue)) {
                        %s
                    }
                }
            }
        }
    `, dgType, objectType, fieldName, fieldsStr)

	vars := map[string]string{
		"$subjectUID": subjectUID,
		"$predicate":  string(predicate),
		"$fieldValue": dgValue,
	}

	resp, err := tx.QueryWithVars(ctx, query, vars)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}

	// Flexibles Unmarshalling um beide Formate zu unterstützen
	var rawResult struct {
		Subject []struct {
			Assertions []struct {
				UID    string          `json:"uid"`
				Object json.RawMessage `json:"object"`
			} `json:"assertions"`
		} `json:"subject"`
	}

	if err := json.Unmarshal(resp.Json, &rawResult); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	if len(rawResult.Subject) == 0 {
		return nil, nil
	}

	// Durchsuche alle Assertions nach passendem Object
	for _, assertion := range rawResult.Subject[0].Assertions {
		if len(assertion.Object) == 0 {
			continue
		}

		// Versuche als Array zu parsen (Standard-Fall)
		var objectArray []T
		if err := json.Unmarshal(assertion.Object, &objectArray); err == nil {
			if len(objectArray) > 0 {
				return &objectArray[0], nil
			}
			continue
		}

		// Versuche als einzelnes Objekt zu parsen (Fallback)
		var singleObject T
		if err := json.Unmarshal(assertion.Object, &singleObject); err == nil {
			return &singleObject, nil
		}
	}

	return nil, nil
}

func GetEntityByFieldInDomain[T any](
	ctx context.Context,
	tx *dgo.Txn,
	domainUID string,
	entityType string,
	fieldName string,
	fieldValue interface{},
) (*T, error) {

	if tx == nil {
		return nil, fmt.Errorf("transaction cannot be nil")
	}

	dgType, dgValue, err := getDgraphTypeAndValue(fieldValue)
	if err != nil {
		return nil, fmt.Errorf("type handling error: %w", err)
	}

	var entityPath string
	switch entityType {
	case "Host":
		entityPath = "has_host"
	case "User":
		entityPath = "has_user"
	case "Service":
		entityPath = "has_host { has_service }"
	default:
		return nil, fmt.Errorf("unsupported entity type for domain: %s", entityType)
	}

	query := fmt.Sprintf(`
	query EntityByFieldInDomain($domainUID: string, $fieldValue: %s) {
		domain(func: uid($domainUID)) @filter(type(Domain)) {
			%s @filter(type(%s) AND eq(%s, $fieldValue)) {
				uid
			}
		}
	}`, dgType, entityPath, entityType, fieldName)

	vars := map[string]string{
		"$domainUID":  domainUID,
		"$fieldValue": dgValue,
	}

	resp, err := tx.QueryWithVars(ctx, query, vars)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}

	type domainWrapper struct {
		Entities []T `json:"has_host"`
	}

	var result struct {
		Domain []domainWrapper `json:"domain"`
	}

	if err := json.Unmarshal(resp.Json, &result); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	if len(result.Domain) == 0 || len(result.Domain[0].Entities) == 0 {
		return nil, nil
	}

	return &result.Domain[0].Entities[0], nil
}

func GetEntityByFieldInProject[T any](
	ctx context.Context,
	tx *dgo.Txn,
	projectID string,
	entityType string,
	fieldName string,
	fieldValue interface{},
	fields []string,
) ([]T, error) {
	if tx == nil {
		return nil, fmt.Errorf("transaction cannot be nil")
	}

	dgType, dgValue, err := getDgraphTypeAndValue(fieldValue)
	if err != nil {
		return nil, fmt.Errorf("type handling error: %w", err)
	}

	var entityPath string
	switch entityType {
	case "Domain":
		entityPath = "has_domain"
	case "Host":
		entityPath = "has_domain { has_host }"
	case "User":
		entityPath = "has_domain { has_user }"
	case "Service":
		entityPath = "has_domain { has_host { has_service } }"
	case "RedPathsModule":
		entityPath = "has_redpaths_modules"
	case "Target":
		entityPath = "has_target"
	default:
		return nil, fmt.Errorf("unknown entity type: %s", entityType)
	}

	fieldsStr := strings.Join(fields, "\n")

	query := fmt.Sprintf(`
        query GetEntityByFieldInProject($fieldValue: %s, $projectID: string) {
            project(func: uid($projectID)) @filter(type(Project)) {
                %s @filter(type(%s) AND eq(%s, $fieldValue)) {
                    %s
                }
            }
        }
    `, dgType, entityPath, entityType, fieldName, fieldsStr)

	vars := map[string]string{
		"$fieldValue": dgValue,
		"$projectID":  projectID,
	}

	res, err := tx.QueryWithVars(ctx, query, vars)
	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}

	var result map[string][]map[string]interface{}
	if err := json.Unmarshal(res.Json, &result); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	projects, ok := result["project"]
	if !ok || len(projects) == 0 {
		return []T{}, nil
	}

	entities := extractEntitiesFromPath(projects[0], entityPath)

	entitiesJSON, err := json.Marshal(entities)
	if err != nil {
		return nil, fmt.Errorf("marshal entities error: %w", err)
	}

	var typedEntities []T
	if err := json.Unmarshal(entitiesJSON, &typedEntities); err != nil {
		return nil, fmt.Errorf("unmarshal to typed entities error: %w", err)
	}

	return typedEntities, nil
}

func extractEntitiesFromPath(project map[string]interface{}, entityPath string) []interface{} {

	parts := strings.Split(entityPath, " { ")
	current := project

	for i, part := range parts {
		part = strings.TrimSpace(strings.Trim(part, "}"))

		value, ok := current[part]
		if !ok {
			return []interface{}{}
		}

		if i == len(parts)-1 {
			if entities, ok := value.([]interface{}); ok {
				return entities
			}
			return []interface{}{}
		}

		if slice, ok := value.([]interface{}); ok && len(slice) > 0 {
			if next, ok := slice[0].(map[string]interface{}); ok {
				current = next
			} else {
				return []interface{}{}
			}
		} else {
			return []interface{}{}
		}
	}

	return []interface{}{}
}

func ExistsByFieldOnParent(
	ctx context.Context,
	tx *dgo.Txn,
	parentUID string,
	parentType string,
	childType string,
	relationName string,
	fieldName string,
	fieldValue interface{},
) (exists bool, childUID string, err error) {
	if tx == nil {
		return false, "", fmt.Errorf("transaction cannot be nil")
	}

	dgType, dgValue, err := getDgraphTypeAndValue(fieldValue)
	if err != nil {
		return false, "", fmt.Errorf("type handling error: %w", err)
	}

	query := fmt.Sprintf(`
        query CheckChildOnParent($parentUID: string, $fieldValue: %s) {
            parent(func: uid($parentUID)) @filter(type(%s)) {
                %s @filter(type(%s) AND eq(%s, $fieldValue)) {
                    uid
                }
            }
        }
    `, dgType, parentType, relationName, childType, fieldName)

	vars := map[string]string{
		"$parentUID":  parentUID,
		"$fieldValue": dgValue,
	}

	res, err := tx.QueryWithVars(ctx, query, vars)
	if err != nil {
		return false, "", fmt.Errorf("query error: %w", err)
	}

	var result map[string][]interface{}
	if err := json.Unmarshal(res.Json, &result); err != nil {
		return false, "", fmt.Errorf("unmarshal error: %w", err)
	}

	parents, ok := result["parent"]
	if !ok || len(parents) == 0 {
		return false, "", nil
	}

	parentMap, ok := parents[0].(map[string]interface{})
	if !ok {
		return false, "", nil
	}

	children, ok := parentMap[relationName].([]interface{})
	if !ok || len(children) == 0 {
		return false, "", nil
	}

	if childMap, ok := children[0].(map[string]interface{}); ok {
		if uid, ok := childMap["uid"].(string); ok {
			return true, uid, nil
		}
	}

	return false, "", nil
}

func hasEntitiesInPath(data interface{}, entityType string) bool {
	obj, ok := data.(map[string]interface{})
	if !ok {
		return false
	}

	for _, value := range obj {
		switch v := value.(type) {
		case []interface{}:
			if len(v) > 0 {
				if firstItem, ok := v[0].(map[string]interface{}); ok {
					if _, hasUID := firstItem["uid"]; hasUID {
						return true
					}
				}
				for _, item := range v {
					if hasEntitiesInPath(item, entityType) {
						return true
					}
				}
			}
		case map[string]interface{}:
			if hasEntitiesInPath(v, entityType) {
				return true
			}
		}
	}
	return false
}

// GetEntitiesWithAssertionsAndEmbeddedRelation traversiert über Assertions und lädt zusätzlich
// eine direkt verbundene Relation in das Object-Entity ein.
// Beispiel: Domain -> (Assertion) -> GPOLink -> (direkte Relation) -> GPO
func GetEntitiesWithAssertionsAndEmbeddedRelation[T any](
	ctx context.Context,
	tx *dgo.Txn,
	subjectUID string,
	predicate core.Predicate,
	objectType string,
	objectFields []string,
	embeddedRelation string, // z.B. "gpo.links_to"
	embeddedType string, // z.B. "GPO"
	embeddedFields []string, // Felder der eingebetteten Entity
	queryName string,
) ([]*res.EntityResult[T], error) {
	if tx == nil {
		return nil, fmt.Errorf("transaction cannot be nil")
	}
	if queryName == "" {
		queryName = "getEntitiesWithAssertionsAndEmbedded"
	}

	// Default object fields
	if len(objectFields) == 0 {
		objectFields = []string{
			"uid",
			"dgraph.type",
		}
	}

	// Default embedded fields
	if len(embeddedFields) == 0 {
		embeddedFields = []string{
			"uid",
			"dgraph.type",
		}
	}

	// Build field strings
	fieldsStr := strings.Join(objectFields, "\n                        ")
	embeddedFieldsStr := strings.Join(embeddedFields, "\n                            ")

	// Optional type filter for the object node
	typeFilter := ""
	if strings.TrimSpace(objectType) != "" {
		typeFilter = fmt.Sprintf("@filter(type(%s))", objectType)
	}

	// Optional type filter for embedded relation
	embeddedTypeFilter := ""
	if strings.TrimSpace(embeddedType) != "" {
		embeddedTypeFilter = fmt.Sprintf("@filter(type(%s))", embeddedType)
	}

	// Build Query with embedded relation
	query := fmt.Sprintf(`
		query %s($subjectUID: string, $predicate: string) {
			subject(func: uid($subjectUID)) {
				assertions: ~assertion.subject @filter(eq(assertion.predicate, $predicate)) {
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
					object: assertion.object %s {
						%s
						%s %s {
							%s
						}
					}
				}
			}
		}
	`, queryName, typeFilter, fieldsStr, embeddedRelation, embeddedTypeFilter, embeddedFieldsStr)

	variables := map[string]string{
		"$subjectUID": subjectUID,
		"$predicate":  string(predicate),
	}

	resp, err := tx.QueryWithVars(ctx, query, variables)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	// Flexibles Unmarshalling mit RawMessage
	var rawResult struct {
		Subject []struct {
			Assertions []struct {
				AssertionUID        string          `json:"assertion_uid"`
				Predicate           string          `json:"assertion.predicate"`
				Method              string          `json:"assertion.method"`
				Source              string          `json:"assertion.source"`
				Confidence          float64         `json:"assertion.confidence"`
				Status              string          `json:"assertion.status"`
				Timestamp           time.Time       `json:"assertion.timestamp"`
				Note                string          `json:"assertion.note"`
				HighValueMarked     bool            `json:"assertion.high_value_marked"`
				HasDiscoveredParent bool            `json:"assertion.has_discovered_parent"`
				Object              json.RawMessage `json:"object"`
			} `json:"assertions"`
		} `json:"subject"`
	}

	if err := json.Unmarshal(resp.Json, &rawResult); err != nil {
		return nil, fmt.Errorf("unmarshal failed: %w", err)
	}

	if len(rawResult.Subject) == 0 {
		log.Printf("[%s] No subject found with UID %s", queryName, subjectUID)
		return []*res.EntityResult[T]{}, nil
	}

	var entityResults []*res.EntityResult[T]

	for _, assertionData := range rawResult.Subject[0].Assertions {
		if len(assertionData.Object) == 0 {
			continue
		}

		// Parse Object flexibel - versuche Array, dann einzelnes Objekt
		var entities []T

		// Versuch 1: Als Array parsen
		if err := json.Unmarshal(assertionData.Object, &entities); err != nil {
			// Versuch 2: Als einzelnes Objekt parsen
			var singleEntity T
			if err := json.Unmarshal(assertionData.Object, &singleEntity); err != nil {
				log.Printf("[%s] warn: failed to unmarshal object: %v\njson: %s",
					queryName, err, string(assertionData.Object))
				continue
			}
			entities = []T{singleEntity}
		}

		// Wenn keine Entities gefunden, überspringe
		if len(entities) == 0 {
			continue
		}

		// Verarbeite jede Entity (normalerweise nur eine)
		for _, entity := range entities {
			// Extract object UID aus der Entity
			var objectUID string
			entityValue := reflect.ValueOf(entity)
			if entityValue.Kind() == reflect.Ptr {
				entityValue = entityValue.Elem()
			}
			if entityValue.Kind() == reflect.Struct {
				uidField := entityValue.FieldByName("UID")
				if uidField.IsValid() && uidField.Kind() == reflect.String {
					objectUID = uidField.String()
				}
			}

			// Build core.Assertion
			assertion := &core.Assertion{
				UID:                 assertionData.AssertionUID,
				Predicate:           core.Predicate(assertionData.Predicate),
				Method:              core.Method(assertionData.Method),
				Source:              assertionData.Source,
				Confidence:          assertionData.Confidence,
				Status:              core.Status(assertionData.Status),
				Timestamp:           assertionData.Timestamp,
				Note:                assertionData.Note,
				MarkedAsHighValue:   assertionData.HighValueMarked,
				HasDiscoveredParent: assertionData.HasDiscoveredParent,
				Subject:             &utils.UIDRef{UID: subjectUID},
				Object:              &utils.UIDRef{UID: objectUID},
			}

			entityResult := &res.EntityResult[T]{
				Entity:     entity,
				Assertions: []*core.Assertion{assertion},
				Metadata: &res.ResultMetadata{
					Source:         assertionData.Source,
					ScanTimestamp:  assertionData.Timestamp,
					EntityCount:    1,
					AssertionCount: 1,
				},
			}

			entityResults = append(entityResults, entityResult)
		}
	}

	log.Printf("[%s] Found %d entities with assertions and embedded relation (subject: %s, predicate: %s)",
		queryName, len(entityResults), subjectUID, predicate)

	return entityResults, nil
}
