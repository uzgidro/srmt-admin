package mongo

import (
	"context"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Repo struct {
	Client *mongo.Client
}

func New(client *mongo.Client) *Repo {
	return &Repo{client}
}

func (r *Repo) Close(ctx context.Context) error {
	return r.Client.Disconnect(ctx)
}
