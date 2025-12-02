package mongo

import (
	"context"
	"errors"
	"fmt"
	"srmt-admin/internal/storage"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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

func (r *Repo) GetLatestStockData(ctx context.Context) (string, error) {
	const op = "storage.mongo.GetLatestStockData"
	return r.getRawJSON(ctx, "stock_data", op)
}

func (r *Repo) SaveSnowData(ctx context.Context, jsonData string) error {
	const op = "storage.mongo.SaveSnowData"
	return r.saveRawJSON(ctx, "modsnow_data", jsonData, op)
}

func (r *Repo) GetLatestSnowData(ctx context.Context) (string, error) {
	const op = "storage.mongo.GetLatestSnowData"
	return r.getRawJSON(ctx, "modsnow_data", op)
}

func (r *Repo) GetDC(ctx context.Context) (string, error) {
	const op = "storage.mongo.GetDC"
	return r.getRawJSON(ctx, "dc", op)
}

func (r *Repo) getRawJSON(ctx context.Context, collectionName, op string) (string, error) {
	collection := r.Client.Database("srmt").Collection(collectionName)

	findOptions := options.FindOne().SetSort(bson.D{{"createdAt", -1}})

	var result struct {
		Data string `bson:"data"`
	}

	err := collection.FindOne(ctx, bson.D{}, findOptions).Decode(&result)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return "", storage.ErrDataNotFound
		}
		return "", fmt.Errorf("%s: failed to find document: %w", op, err)
	}

	return result.Data, nil
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
