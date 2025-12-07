package redpaths

import (
	"RedPaths-server/internal/db"
	rprepo "RedPaths-server/internal/repository/redpaths"
	"RedPaths-server/pkg/model/redpaths"
	"context"
	"log"

	"gorm.io/gorm"
)

type CollectionService struct {
	db                     *gorm.DB
	redPathsModuleRepo     rprepo.RedPathsModuleRepository
	redPathsCollectionRepo rprepo.RedPathsCollectionRepository
}

func NewCollectionService(postgresCon *gorm.DB) (*CollectionService, error) {

	return &CollectionService{
		db:                     postgresCon,
		redPathsModuleRepo:     rprepo.NewPostgresRedPathsModuleRepository(),
		redPathsCollectionRepo: rprepo.NewPostgresRedPathsCollectionRepository(),
	}, nil
}

func (s *CollectionService) GetAllCollectionsWithModules(ctx context.Context) ([]*redpaths.Collection, error) {
	return db.ExecutePostgresRead(ctx, s.db, func(db *gorm.DB) ([]*redpaths.Collection, error) {
		collections, err := s.redPathsCollectionRepo.GetAll(ctx, db)
		if err != nil {
			return nil, err
		}

		for _, collection := range collections {
			modules, err := s.redPathsCollectionRepo.GetModulesForCollection(ctx, db, collection.ID)
			if err != nil {
				return nil, err
			}

			collection.Modules = modules
		}

		if len(collections) == 0 {
			log.Println("no collections found in database")
		}
		return collections, nil
	})
}
