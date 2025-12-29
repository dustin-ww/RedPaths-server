package active_directory

import (
	"RedPaths-server/internal/db"
	"RedPaths-server/internal/repository/active_directory"
	"RedPaths-server/internal/utils"
	active_directory2 "RedPaths-server/pkg/model/active_directory"
	"context"
	"fmt"
	"log"

	"github.com/dgraph-io/dgo/v210"
)

type UserService struct {
	userRepo    active_directory.UserRepository
	projectRepo active_directory.ProjectRepository
	db          *dgo.Dgraph
}

func NewUserService(dgraphCon *dgo.Dgraph) (*UserService, error) {
	userRepo := active_directory.NewDgraphUserRepository(dgraphCon)
	projectRepo := active_directory.NewDgraphProjectRepository(dgraphCon)

	return &UserService{
		db:          dgraphCon,
		projectRepo: projectRepo,
		userRepo:    userRepo,
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
