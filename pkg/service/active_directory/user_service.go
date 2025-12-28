package active_directory

import (
	"RedPaths-server/internal/db"
	"RedPaths-server/internal/repository/active_directory"
	"RedPaths-server/pkg/model"
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

func (s *UserService) Create(ctx context.Context, user *model.ADUser, projectUID string, actor string) (*model.ADUser, error) {
	var createdUser *model.ADUser
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
