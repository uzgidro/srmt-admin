package redis

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

// buildAlarmKey builds the Redis key for alarm state tracking
// Format: alarm:active:{station_id}:{device_id}
func (r *Repo) buildAlarmKey(stationID int64, deviceID string) string {
	return fmt.Sprintf("alarm:active:%d:%s", stationID, deviceID)
}

// GetActiveShutdown returns the shutdown ID for an active alarm
// Returns 0 if no active shutdown exists
func (r *Repo) GetActiveShutdown(ctx context.Context, stationID int64, deviceID string) (int64, error) {
	key := r.buildAlarmKey(stationID, deviceID)

	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, nil // No active shutdown
		}
		return 0, fmt.Errorf("get alarm state %s: %w", key, err)
	}

	shutdownID, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse shutdown id %s: %w", val, err)
	}

	return shutdownID, nil
}

// SetActiveShutdown stores the shutdown ID for an active alarm
// No TTL - key is removed only when alarm clears
func (r *Repo) SetActiveShutdown(ctx context.Context, stationID int64, deviceID string, shutdownID int64) error {
	key := r.buildAlarmKey(stationID, deviceID)

	err := r.client.Set(ctx, key, shutdownID, 0).Err() // 0 = no expiration
	if err != nil {
		return fmt.Errorf("set alarm state %s: %w", key, err)
	}

	return nil
}

// ClearActiveShutdown removes the active alarm record
func (r *Repo) ClearActiveShutdown(ctx context.Context, stationID int64, deviceID string) error {
	key := r.buildAlarmKey(stationID, deviceID)

	err := r.client.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("clear alarm state %s: %w", key, err)
	}

	return nil
}
