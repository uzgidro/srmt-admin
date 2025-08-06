package mongo

import (
	"context"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/v2/bson"
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

func (r *Repo) SaveStockData(ctx context.Context, jsonData string) error {
	const op = "storage.mongo.SaveStockData"

	var doc bson.M
	if err := json.Unmarshal([]byte(jsonData), &doc); err != nil {
		return fmt.Errorf("%s: failed to unmarshal json to bson: %w", op, err)
	}

	collection := r.Client.Database("srmt_db").Collection("stock_data")

	_, err := collection.InsertOne(ctx, doc)
	if err != nil {
		return fmt.Errorf("%s: failed to insert document: %w", op, err)
	}

	return nil
}
