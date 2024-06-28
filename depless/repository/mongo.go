package repository

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Cfg struct {
	Host string
	User string
	Pass string
	Name string
}

func (c Cfg) Creds() *options.Credential {
	if c.Pass != "" && c.User != "" {
		return &options.Credential{
			Username:    c.User,
			Password:    c.Pass,
			PasswordSet: true,
		}
	}

	return nil
}

type MongoStorage struct {
	cfg Cfg
	*mongo.Database
}

func New(ctx context.Context, cfg Cfg) *MongoStorage {
	r := &MongoStorage{
		cfg: cfg,
	}

	r.connect(ctx)
	go r.handleReconnect(ctx)

	return r
}

func (r *MongoStorage) connect(ctx context.Context) {
	opts := options.Client().
		ApplyURI(r.cfg.Host).
		SetConnectTimeout(time.Second * 3).
		SetTimeout(time.Second * 3)

	if creds := r.cfg.Creds(); creds != nil {
		opts.SetAuth(*creds)
	}
	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		panic(err)
	}

	r.Database = client.Database(r.cfg.Name)
}

func (r *MongoStorage) handleReconnect(ctx context.Context) {
	t := time.NewTimer(0)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
		if err := r.Database.Client().Ping(ctx, readpref.Primary()); err != nil {
			r.connect(ctx)
		}

		t.Reset(5 * time.Second)
	}
}
