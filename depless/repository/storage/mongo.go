package storage

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"

	"go.mongodb.org/mongo-driver/mongo"
)

type Storage struct {
	apps        *mongo.Collection
	deployments *mongo.Collection
}

const (
	apps        = "apps"
	deployments = "deployments"
)

func New(db *mongo.Database) *Storage {
	return &Storage{
		apps:        db.Collection(apps),
		deployments: db.Collection(deployments),
	}
}

func NewDeployment(appID uuid.UUID, version string, buildBytes []byte, archivePath string, repoURI string) *Deployment {
	return &Deployment{
		ID:          uuid.New(),
		AppID:       appID,
		Version:     version,
		BuildBytes:  buildBytes,
		ArchivePath: archivePath,
		RepoURI:     repoURI,
		Hash:        sum(buildBytes),
	}
}

func (d *Deployment) sumUpdate() {
	d.Hash = sum(d.BuildBytes)
}

func sum(b []byte) string {
	s := md5.Sum(b)

	return hex.EncodeToString(s[:])
}

func (s *Storage) CreateApp(ctx context.Context, app *App) error {
	_, err := s.apps.InsertOne(ctx, app)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) GetAppByID(ctx context.Context, id uuid.UUID) (App, error) {
	var app App
	err := s.apps.FindOne(ctx, bson.D{
		bson.E{
			"_id", id,
		},
	}).Decode(&app)
	if err != nil {
		return app, err
	}

	return app, nil
}

func (s *Storage) CreateDeployment(ctx context.Context, deployment *Deployment) error {
	_, err := s.deployments.InsertOne(ctx, deployment)
	if err != nil {
		return err
	}

	return nil
}

func (s *Storage) GetDeploymentByID(ctx context.Context, id uuid.UUID) (Deployment, error) {
	var deployment Deployment
	err := s.deployments.FindOne(ctx, bson.D{
		bson.E{
			"_id", id,
		},
	}).Decode(&deployment)
	if err != nil {
		return deployment, err
	}

	return deployment, nil
}

type updateFn[T any] func(v *T) error

func (s *Storage) UpdateApp(ctx context.Context, id uuid.UUID, upd func(app *App) error) error {
	app, err := s.GetAppByID(ctx, id)
	if err != nil {
		return err
	}
	if err = upd(&app); err != nil {
		return err
	}

	_, err = s.apps.UpdateByID(ctx, id, app)

	return err
}
func (s *Storage) UpdateDeployment(ctx context.Context, id uuid.UUID, upd func(app *Deployment) error) error {
	dep, err := s.GetDeploymentByID(ctx, id)
	if err != nil {
		return err
	}
	newDep := dep
	if err = upd(&newDep); err != nil {
		return err
	}
	if !bytes.Equal(newDep.BuildBytes, dep.BuildBytes) {
		newDep.sumUpdate()
	}
	_, err = s.deployments.UpdateByID(ctx, id, newDep)

	return err
}
