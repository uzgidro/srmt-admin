package mongo

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"srmt-admin/internal/config"
)

func New(ctx context.Context, cfg config.Mongo) (*mongo.Client, error) {
	const op = "storage.driver.mongo.New"

	uri := fmt.Sprintf("mongodb://%s:%s", cfg.Host, cfg.Port)
	clientOptions := options.Client().ApplyURI(uri)

	if cfg.Username != "" && cfg.Password != "" {
		cred := options.Credential{
			Username: cfg.Username,
			Password: cfg.Password,
		}
		if cfg.AuthSource != "" {
			cred.AuthSource = cfg.AuthSource
		}
		clientOptions.SetAuth(cred)
	}

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to connect to mongo: %w", op, err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		// Если пинг не прошел, пытаемся закрыть соединение, чтобы очистить ресурсы.
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("%s: failed to ping mongo: %w", op, err)
	}

	return client, nil
}
