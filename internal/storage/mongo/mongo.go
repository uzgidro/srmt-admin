package mongo

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"time"
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
	return r.saveRawJSON(ctx, "stock_data", jsonData, op)
}

func (r *Repo) SaveSnowData(ctx context.Context, jsonData string) error {
	const op = "storage.mongo.SaveSnowData"
	return r.saveRawJSON(ctx, "modsnow_data", jsonData, op)
}

func (r *Repo) saveRawJSON(ctx context.Context, collectionName, jsonData, op string) error {
	collection := r.Client.Database("srmt").Collection(collectionName)

	doc := bson.D{
		{Key: "data", Value: jsonData},
		{Key: "createdAt", Value: time.Now()},
	}

	if _, err := collection.InsertOne(ctx, doc); err != nil {
		return fmt.Errorf("%s: failed to insert raw json document: %w", op, err)
	}

	return nil
}
