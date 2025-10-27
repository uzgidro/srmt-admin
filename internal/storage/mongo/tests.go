package mongo

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"srmt-admin/internal/lib/model/test"
	"srmt-admin/internal/storage"
)

func (r *Repo) GetRandomGidroTest(ctx context.Context) (*test.GidroTest, error) {
	const op = "storage.mongo.GetRandomGidroTest"

	collection := r.Client.Database("gidro_quest").Collection("test")

	pipeline := mongo.Pipeline{
		{{Key: "$sample", Value: bson.D{{Key: "size", Value: 1}}}},
	}

	cursor, err := collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to execute aggregation: %w", op, err)
	}
	defer cursor.Close(ctx)

	if !cursor.Next(ctx) {
		if err := cursor.Err(); err != nil {
			return nil, fmt.Errorf("%s: cursor error: %w", op, err)
		}
		return nil, storage.ErrDataNotFound
	}

	var test test.GidroTest
	if err := cursor.Decode(&test); err != nil {
		return nil, fmt.Errorf("%s: failed to decode document: %w", op, err)
	}

	return &test, nil
}
