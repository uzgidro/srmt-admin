package alarm

import (
	"context"
	"log/slog"
	"time"

	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/model/asutp"
)

// ShutdownManager provides methods to manage shutdowns
type ShutdownManager interface {
	AddShutdown(ctx context.Context, req dto.AddShutdownRequest) (int64, error)
	EditShutdown(ctx context.Context, id int64, req dto.EditShutdownRequest) error
}

// StateTracker provides methods to track alarm state in Redis
type StateTracker interface {
	// GetActiveShutdown returns the shutdown ID for active alarm, 0 if none
	GetActiveShutdown(ctx context.Context, stationID int64, deviceID string) (int64, error)
	// SetActiveShutdown stores the shutdown ID for an active alarm
	SetActiveShutdown(ctx context.Context, stationID int64, deviceID string, shutdownID int64) error
	// ClearActiveShutdown removes the active alarm record
	ClearActiveShutdown(ctx context.Context, stationID int64, deviceID string) error
}

// Processor handles automatic shutdown creation based on ASUTP alarms
type Processor struct {
	shutdownRepo ShutdownManager
	stateTracker StateTracker
	log          *slog.Logger
}

// NewProcessor creates a new alarm processor
func NewProcessor(shutdownRepo ShutdownManager, stateTracker StateTracker, log *slog.Logger) *Processor {
	return &Processor{
		shutdownRepo: shutdownRepo,
		stateTracker: stateTracker,
		log:          log,
	}
}

// ProcessEnvelope processes telemetry envelope for alarms
// Creates shutdowns when alarms trigger and closes them when alarms clear
func (p *Processor) ProcessEnvelope(ctx context.Context, stationDBID int64, env *asutp.Envelope) error {
	const op = "alarm.Processor.ProcessEnvelope"

	log := p.log.With(
		slog.String("op", op),
		slog.Int64("station_id", stationDBID),
		slog.String("device_id", env.DeviceID),
	)

	// Detect triggered alarms
	triggeredAlarms := DetectTriggeredAlarms(env.Values)

	// Get current active shutdown from Redis
	activeShutdownID, err := p.stateTracker.GetActiveShutdown(ctx, stationDBID, env.DeviceID)
	if err != nil {
		// Log warning but continue - better to create duplicate than miss
		log.Warn("failed to get active shutdown from Redis, will create if needed", "error", err)
		activeShutdownID = 0
	}

	if len(triggeredAlarms) > 0 {
		// Alarms are active
		if activeShutdownID == 0 {
			// No active shutdown - create new one
			shutdownID, err := p.createShutdown(ctx, stationDBID, env, triggeredAlarms)
			if err != nil {
				log.Error("failed to create shutdown", "error", err)
				return nil // Don't block telemetry saving
			}

			// Store active shutdown in Redis
			if err := p.stateTracker.SetActiveShutdown(ctx, stationDBID, env.DeviceID, shutdownID); err != nil {
				log.Warn("failed to store active shutdown in Redis", "shutdown_id", shutdownID, "error", err)
			}

			log.Info("created shutdown for alarms",
				"shutdown_id", shutdownID,
				"alarms_count", len(triggeredAlarms),
			)
		}
		// If activeShutdownID != 0, shutdown already exists - do nothing
	} else {
		// No alarms active
		if activeShutdownID != 0 {
			// Close the active shutdown
			if err := p.closeShutdown(ctx, activeShutdownID, env.Timestamp); err != nil {
				log.Error("failed to close shutdown", "shutdown_id", activeShutdownID, "error", err)
				return nil // Don't block telemetry saving
			}

			// Clear from Redis
			if err := p.stateTracker.ClearActiveShutdown(ctx, stationDBID, env.DeviceID); err != nil {
				log.Warn("failed to clear active shutdown from Redis", "shutdown_id", activeShutdownID, "error", err)
			}

			log.Info("closed shutdown (alarms cleared)", "shutdown_id", activeShutdownID)
		}
		// If activeShutdownID == 0, no active shutdown - nothing to do
	}

	return nil
}

// createShutdown creates a new shutdown record
func (p *Processor) createShutdown(ctx context.Context, stationDBID int64, env *asutp.Envelope, alarms []AlarmSignal) (int64, error) {
	reason := FormatReason(env.DeviceID, alarms)

	req := dto.AddShutdownRequest{
		OrganizationID:  stationDBID,
		StartTime:       env.Timestamp,
		EndTime:         nil, // Open-ended
		Reason:          &reason,
		CreatedByUserID: SystemUserID,
	}

	return p.shutdownRepo.AddShutdown(ctx, req)
}

// closeShutdown sets the end time on an existing shutdown
func (p *Processor) closeShutdown(ctx context.Context, shutdownID int64, endTime time.Time) error {
	req := dto.EditShutdownRequest{
		EndTime: &endTime,
	}

	return p.shutdownRepo.EditShutdown(ctx, shutdownID, req)
}
