package upsert

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	reservoirdata "srmt-admin/internal/lib/model/reservoir-data"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/storage"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type Response struct {
	resp.Response
	ProcessedCount int `json:"processed_count"`
}

type ReservoirDataUpserter interface {
	UpsertReservoirData(ctx context.Context, data []reservoirdata.ReservoirDataItem, userID int64) error
}

func New(log *slog.Logger, upserter ReservoirDataUpserter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.reservoir-data.upsert.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Get authenticated user ID
		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		// Parse request body (expecting array directly)
		var data []reservoirdata.ReservoirDataItem
		if err := render.DecodeJSON(r.Body, &data); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		// Validate array is not empty
		if len(data) == 0 {
			log.Warn("empty data array received")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Data array cannot be empty"))
			return
		}

		// Validate each item and date format
		validate := validator.New()
		for i, item := range data {
			// Validate struct
			if err := validate.Struct(item); err != nil {
				var vErrs validator.ValidationErrors
				errors.As(err, &vErrs)
				log.Error("validation failed", sl.Err(err), slog.Int("item_index", i))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.ValidationErrors(vErrs))
				return
			}

			// Validate date format
			if _, err := time.Parse("2006-01-02", item.Date); err != nil {
				log.Error("invalid date format", sl.Err(err), slog.Int("item_index", i), slog.String("date", item.Date))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest(fmt.Sprintf("Invalid date format at index %d. Expected YYYY-MM-DD", i)))
				return
			}
		}

		// Upsert reservoir data
		err = upserter.UpsertReservoirData(r.Context(), data, userID)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("foreign key violation - organization not found")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("One or more organizations not found"))
				return
			}
			log.Error("failed to upsert reservoir data", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to save reservoir data"))
			return
		}

		log.Info("reservoir data upserted successfully",
			slog.Int("count", len(data)),
			slog.Int64("user_id", userID),
		)

		render.Status(r, http.StatusOK)
		render.JSON(w, r, Response{
			Response:       resp.OK(),
			ProcessedCount: len(data),
		})
	}
}
