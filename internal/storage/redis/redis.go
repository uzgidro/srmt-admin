package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"srmt-admin/internal/lib/model/asutp"
)

type Repo struct {
	client *redis.Client
	ttl    time.Duration
}

func New(client *redis.Client, ttlSeconds int) *Repo {
	return &Repo{
		client: client,
		ttl:    time.Duration(ttlSeconds) * time.Second,
	}
}

func (r *Repo) buildKey(stationDBID int64, deviceID string) string {
	return fmt.Sprintf("asutp:%d:%s", stationDBID, deviceID)
}

func (r *Repo) buildPattern(stationDBID int64) string {
	return fmt.Sprintf("asutp:%d:*", stationDBID)
}

// SaveTelemetry saves device telemetry data
func (r *Repo) SaveTelemetry(ctx context.Context, stationDBID int64, env *asutp.Envelope) error {
	key := r.buildKey(stationDBID, env.DeviceID)

	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	return r.client.Set(ctx, key, data, r.ttl).Err()
}

// GetDeviceTelemetry retrieves telemetry for a specific device
func (r *Repo) GetDeviceTelemetry(ctx context.Context, stationDBID int64, deviceID string) (*asutp.Envelope, error) {
	key := r.buildKey(stationDBID, deviceID)

	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("get key %s: %w", key, err)
	}

	var env asutp.Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("unmarshal envelope: %w", err)
	}

	return &env, nil
}

// GetStationTelemetry retrieves telemetry for all devices of a station
func (r *Repo) GetStationTelemetry(ctx context.Context, stationDBID int64) ([]*asutp.Envelope, error) {
	pattern := r.buildPattern(stationDBID)

	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("get keys for pattern %s: %w", pattern, err)
	}

	if len(keys) == 0 {
		return []*asutp.Envelope{}, nil
	}

	values, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("mget keys: %w", err)
	}

	result := make([]*asutp.Envelope, 0, len(values))
	for _, val := range values {
		if val == nil {
			continue
		}

		strVal, ok := val.(string)
		if !ok {
			continue
		}

		var env asutp.Envelope
		if err := json.Unmarshal([]byte(strVal), &env); err != nil {
			continue
		}

		result = append(result, &env)
	}

	return result, nil
}
