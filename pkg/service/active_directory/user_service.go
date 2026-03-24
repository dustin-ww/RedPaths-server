package active_directory

import (
	"RedPaths-server/internal/db"
	"RedPaths-server/internal/repository/active_directory"
	engine2 "RedPaths-server/internal/repository/redpaths/engine"
	"RedPaths-server/internal/repository/util/dgraph"
	"RedPaths-server/internal/utils"
	active_directory2 "RedPaths-server/pkg/model/active_directory"
	"RedPaths-server/pkg/model/core"
	"RedPaths-server/pkg/model/core/res"
	utils2 "RedPaths-server/pkg/model/utils"
	engine3 "RedPaths-server/pkg/service/catalog"
	engine4 "RedPaths-server/pkg/service/upsert"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/dgraph-io/dgo/v210"
)

type UserService struct {
	userRepo       active_directory.UserRepository
	projectRepo    active_directory.ProjectRepository
	assertionRepo  engine2.AssertionRepository
	catalogService *engine3.CatalogService
	db             *dgo.Dgraph
}

func NewUserService(dgraphCon *dgo.Dgraph) (*UserService, error) {
	userRepo := active_directory.NewDgraphUserRepository(dgraphCon)
	projectRepo := active_directory.NewDgraphProjectRepository(dgraphCon)
	assertionRepo := engine2.NewDgraphAssertionRepository(dgraphCon)
	catalogService := engine3.NewCatalogService(dgraphCon)

	return &UserService{
		db:             dgraphCon,
		projectRepo:    projectRepo,
		userRepo:       userRepo,
		assertionRepo:  assertionRepo,
		catalogService: catalogService,
	}, nil
}

func (s *UserService) Create(ctx context.Context, user *active_directory2.User, projectUID string, actor string) (*active_directory2.User, error) {
	var createdUser *active_directory2.User
	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {

		var err error
		createdUser, err = s.userRepo.Create(ctx, tx, user, actor)
		log.Printf("Creating User with new uid %s and unknown domain in project with uid %s", createdUser.UID, projectUID)
		if err != nil {
			return fmt.Errorf("failed to create host: %w", err)
		}

		if err := s.projectRepo.AddUser(ctx, tx, projectUID, createdUser.UID); err != nil {
			return fmt.Errorf("failed to reverse link unknown domain user to project: %w", err)
		}

		return nil
	})
	return createdUser, err
}

// -----------------------------------------------------------------------------
// UpsertUser
// -----------------------------------------------------------------------------

func (s *UserService) UpsertUser(
	ctx context.Context,
	input engine4.Input[*active_directory2.User],
) (*res.EntityResult[*active_directory2.User], error) {

	subjectUID, subjectType, hasParent := input.Resolved()

	var result *res.EntityResult[*active_directory2.User]

	err := db.ExecuteInTransaction(ctx, s.db, func(tx *dgo.Txn) error {
		// --- Existence Check ---
		existence, err := s.userRepo.FindExisting(ctx, tx, input.ProjectUID, input.Entity)
		if err != nil {
			return fmt.Errorf("existence check failed: %w", err)
		}

		filters := active_directory.BuildUserFilter(input.Entity)
		var actualUser *active_directory2.User

		switch existence.FoundVia {

		// --- Case 1: Not found → create new ---
		case dgraph.ExistenceSourceNotFound:
			createdUser, err := s.userRepo.Create(ctx, tx, input.Entity, input.Actor)
			if err != nil {
				return fmt.Errorf("creating user: %w", err)
			}
			actualUser = createdUser
			log.Printf("[UpsertUser] Created uid=%s name=%s", actualUser.UID, actualUser.Name)

		case dgraph.ExistenceSourceHierarchy,
			dgraph.ExistenceSourceProject:

			best := dgraph.BestCandidate(existence.Entities, filters, 0.5)

			if best == nil {
				// Candidates found but score too low → create new
				createdUser, err := s.userRepo.Create(ctx, tx, input.Entity, input.Actor)
				if err != nil {
					return fmt.Errorf("creating user (low score): %w", err)
				}
				actualUser = createdUser
				log.Printf("[UpsertUser] Low score, created uid=%s", actualUser.UID)

			} else if best.Score >= 0.8 {
				// --- Case 2: High Confidence → Merge ---
				mergeFields := buildUserMergeFields(
					best.Result.Entity,
					input.Entity,
					input.AssertionCtx.GetConfidence(),
				)
				updated, err := s.userRepo.UpdateUser(
					ctx, tx,
					best.Result.Entity.UID,
					input.Actor,
					mergeFields,
				)
				if err != nil {
					return fmt.Errorf("merging user: %w", err)
				}
				actualUser = updated
				log.Printf("[UpsertUser] Merged uid=%s score=%.2f",
					actualUser.UID, best.Score)

			} else {
				// --- Case 3: Medium Confidence (0.5–0.8) → Possible Duplicate ---
				log.Printf("[UpsertUser] Possible duplicate uid=%s score=%.2f",
					best.Result.Entity.UID, best.Score)

				duplicateAssertion := &core.Assertion{
					Predicate:  core.PredicatePossibleDuplicate,
					Method:     core.MethodInferred,
					Source:     input.Actor,
					Confidence: best.Score,
					Status:     core.StatusTentative,
					Timestamp:  time.Now(),
					Note: fmt.Sprintf(
						"Possible duplicate detected with score %.2f — manual review required",
						best.Score,
					),
					HasDiscoveredParent: false,
					MarkedAsHighValue:   false,
					Subject:             &utils2.UIDRef{UID: best.Result.Entity.UID, Type: "User"},
					Object:              &utils2.UIDRef{UID: input.Entity.UID, Type: "User"},
				}

				if _, err := s.assertionRepo.Create(ctx, tx, duplicateAssertion); err != nil {
					return fmt.Errorf("creating duplicate assertion: %w", err)
				}

				result = best.Result
				return nil
			}

		default:
			return fmt.Errorf("unhandled existence state: %s", existence.FoundVia)
		}

		// --- Create assertion (for both Create and Merge) ---
		assertionSchema := &core.Assertion{
			Predicate:           core.PredicateContains,
			Method:              core.MethodDirectAdd,
			Source:              input.Actor,
			Confidence:          input.AssertionCtx.GetConfidence(),
			Status:              core.StatusValidated,
			Timestamp:           time.Now(),
			HasDiscoveredParent: hasParent,
			MarkedAsHighValue:   input.AssertionCtx.IsHighValue(),
			Subject:             &utils2.UIDRef{UID: subjectUID, Type: subjectType},
			Object:              &utils2.UIDRef{UID: actualUser.UID, Type: "User"},
		}

		createdAssertion, err := s.assertionRepo.Create(ctx, tx, assertionSchema)
		if err != nil {
			return fmt.Errorf("creating assertion: %w", err)
		}

		result = &res.EntityResult[*active_directory2.User]{
			Entity:     actualUser,
			Assertions: []*core.Assertion{createdAssertion},
			Metadata: &res.ResultMetadata{
				Source:         input.Actor,
				ScanTimestamp:  time.Now(),
				EntityCount:    1,
				AssertionCount: 1,
			},
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("UpsertUser failed: %w", err)
	}

	// --- Catalog Integration (outside the transaction) ---
	if result == nil || len(result.Assertions) == 0 {
		return result, nil
	}

	_, catalogErr := engine3.AddToCatalog(
		ctx,
		s.catalogService,
		input.ProjectUID,
		result.Entity.UID,
		"User",
		result.Assertions[0],
		input.Actor,
	)
	if catalogErr != nil {
		log.Printf("[UpsertUser] Warning: failed to add user %s to catalog: %v",
			result.Entity.UID, catalogErr)
	}

	if hasParent {
		promoteErr := engine3.PromoteInCatalog(
			ctx,
			s.catalogService,
			input.ProjectUID,
			result.Entity.UID,
			"User",
			core.PredicateContains,
			input.Actor,
		)
		if promoteErr != nil {
			log.Printf("[UpsertUser] Warning: failed to promote user %s in catalog: %v",
				result.Entity.UID, promoteErr)
		}
	}

	return result, nil
}

// -----------------------------------------------------------------------------
// buildUserMergeFields
// -----------------------------------------------------------------------------

func buildUserMergeFields(
	existing *active_directory2.User,
	incoming *active_directory2.User,
	incomingConfidence float64,
) map[string]interface{} {
	fields := map[string]interface{}{
		"last_seen_at": time.Now(),
	}

	// Name: only overwrite if existing is empty
	if incoming.Name != "" && existing.Name == "" {
		fields["security_principal.name"] = incoming.Name
	}

	// SID: truly unique, overwrite only if existing is empty
	if incoming.SID != "" && existing.SID == "" {
		fields["security_principal.sid"] = incoming.SID
	}

	// SAMAccountName: overwrite if incoming set and existing empty
	if incoming.SAMAccountName != "" && existing.SAMAccountName == "" {
		fields["user.sam_account_name"] = incoming.SAMAccountName
	}

	// UPN: overwrite if incoming set and existing empty
	if incoming.UPN != "" && existing.UPN == "" {
		fields["user.upn"] = incoming.UPN
	}

	// Boolean flags: once set, they should not be revoked by lower-confidence data
	if incoming.IsDomainAdmin && !existing.IsDomainAdmin {
		fields["user.is_domain_admin"] = true
	}
	if incoming.IsLocalAdmin && !existing.IsLocalAdmin {
		fields["user.is_local_admin"] = true
	}

	// Kerberos attack surface: always reflect the current state if confidence is high
	if incomingConfidence >= 0.8 {
		fields["user.kerberoastable"] = incoming.Kerberoastable
		fields["user.asrep_roastable"] = incoming.ASREPRoastable
		fields["user.is_disabled"] = incoming.IsDisabled
		fields["user.is_locked"] = incoming.IsLocked
	}

	return fields
}

// -----------------------------------------------------------------------------
// UpdateUser
// -----------------------------------------------------------------------------

func (s *UserService) UpdateUser(ctx context.Context, uid, actor string, fields map[string]interface{}) (*active_directory2.User, error) {
	if uid == "" {
		return nil, utils.ErrUIDRequired
	}

	/*allowed := map[string]bool{"name": true, "description": true}
	protected := map[string]bool{"uid": true, "created_at": true, "updated_at": true, "type": true}

	for field := range fields {
		if protected[field] {
			return nil, fmt.Errorf("%w: %s", utils.ErrFieldProtected, field)
		}
		if !allowed[field] {
			return nil, fmt.Errorf("%w: %s", utils.ErrFieldNotAllowed, field)
		}
	}*/

	return db.ExecuteInTransactionWithResult[*active_directory2.User](ctx, s.db, func(tx *dgo.Txn) (*active_directory2.User, error) {
		return s.userRepo.UpdateUser(ctx, tx, uid, actor, fields)
	})
}
