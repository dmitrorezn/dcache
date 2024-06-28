package storage

import "github.com/google/uuid"

type Deployment struct {
	ID          uuid.UUID `bson:"_id"`
	AppID       uuid.UUID `bson:"app_id"`
	Version     string    `bson:"version"`
	Hash        string    `bson:"hash"`
	BuildBytes  []byte    `bson:"build_bytes"`
	ArchivePath string    `bson:"archive_path"`
	RepoURI     string    `bson:"repo_uri"`
}

type App struct {
	ID              uuid.UUID `bson:"_id"`
	Name            string    `bson:"name"`
	OwnerID         uuid.UUID `bson:"owner_id"`
	Url             string    `bson:"url"`
	CurrentDeployID uuid.UUID `bson:"deploy_id"`
}

type User struct {
	ID    uuid.UUID
	Login string
}

func NewUser(Login string) *User {
	return &User{
		ID:    uuid.New(),
		Login: Login,
	}
}

func NewApp(user *User, name string) *App {
	return &App{
		ID:              uuid.New(),
		OwnerID:         user.ID,
		Name:            name,
		CurrentDeployID: uuid.UUID{},
	}
}

func (a *App) Deploy(d *Deployment) {
	a.CurrentDeployID = d.ID
}

func (a *App) SetURL(url string) {
	a.Url = url
}
func (e *App) HasActiveDeploy() bool {
	return e.CurrentDeployID.String() != "00000000-0000-0000-0000-000000000000"
}
