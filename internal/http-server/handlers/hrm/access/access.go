package access

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto/hrm"
	"srmt-admin/internal/lib/logger/sl"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
	"srmt-admin/internal/storage"
)

// --- Repository Interfaces ---

type ZoneRepository interface {
	AddAccessZone(ctx context.Context, req hrm.AddAccessZoneRequest) (int, error)
	GetAccessZoneByID(ctx context.Context, id int) (*hrmmodel.AccessZone, error)
	GetAccessZones(ctx context.Context, filter hrm.AccessZoneFilter) ([]*hrmmodel.AccessZone, error)
	EditAccessZone(ctx context.Context, id int, req hrm.EditAccessZoneRequest) error
	DeleteAccessZone(ctx context.Context, id int) error
}

type CardRepository interface {
	AddAccessCard(ctx context.Context, req hrm.AddAccessCardRequest) (int64, error)
	GetAccessCardByID(ctx context.Context, id int64) (*hrmmodel.AccessCard, error)
	GetAccessCards(ctx context.Context, filter hrm.AccessCardFilter) ([]*hrmmodel.AccessCard, error)
	EditAccessCard(ctx context.Context, id int64, req hrm.EditAccessCardRequest) error
	DeleteAccessCard(ctx context.Context, id int64) error
	DeactivateAccessCard(ctx context.Context, id int64, reason string) error
}

type CardZoneAccessRepository interface {
	AddCardZoneAccess(ctx context.Context, req hrm.AddCardZoneAccessRequest, grantedBy *int64) (int64, error)
	GetCardZoneAccess(ctx context.Context, filter hrm.CardZoneAccessFilter) ([]*hrmmodel.CardZoneAccess, error)
	DeleteCardZoneAccess(ctx context.Context, id int64) error
}

type AccessLogRepository interface {
	AddAccessLog(ctx context.Context, req hrm.AddAccessLogRequest) (int64, error)
	GetAccessLogs(ctx context.Context, filter hrm.AccessLogFilter) ([]*hrmmodel.AccessLog, error)
}

// IDResponse represents a response with ID
type IDResponse struct {
	resp.Response
	ID int64 `json:"id"`
}

// --- Zone Handlers ---

func GetZones(log *slog.Logger, repo ZoneRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.access.GetZones"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.AccessZoneFilter
		q := r.URL.Query()

		if building := q.Get("building"); building != "" {
			filter.Building = &building
		}

		if secLevelStr := q.Get("security_level"); secLevelStr != "" {
			val, err := strconv.Atoi(secLevelStr)
			if err != nil {
				log.Warn("invalid 'security_level' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'security_level' parameter"))
				return
			}
			filter.SecurityLevel = &val
		}

		if isActiveStr := q.Get("is_active"); isActiveStr != "" {
			val := isActiveStr == "true"
			filter.IsActive = &val
		}

		zones, err := repo.GetAccessZones(r.Context(), filter)
		if err != nil {
			log.Error("failed to get access zones", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve access zones"))
			return
		}

		log.Info("successfully retrieved access zones", slog.Int("count", len(zones)))
		render.JSON(w, r, zones)
	}
}

func GetZoneByID(log *slog.Logger, repo ZoneRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.access.GetZoneByID"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		zone, err := repo.GetAccessZoneByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("access zone not found", slog.Int("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Access zone not found"))
				return
			}
			log.Error("failed to get access zone", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve access zone"))
			return
		}

		render.JSON(w, r, zone)
	}
}

func AddZone(log *slog.Logger, repo ZoneRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.access.AddZone"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddAccessZoneRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		id, err := repo.AddAccessZone(r.Context(), req)
		if err != nil {
			log.Error("failed to add access zone", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add access zone"))
			return
		}

		log.Info("access zone added", slog.Int("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: int64(id)})
	}
}

func EditZone(log *slog.Logger, repo ZoneRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.access.EditZone"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.EditAccessZoneRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.EditAccessZone(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("access zone not found", slog.Int("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Access zone not found"))
				return
			}
			log.Error("failed to update access zone", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update access zone"))
			return
		}

		log.Info("access zone updated", slog.Int("id", id))
		render.JSON(w, r, resp.OK())
	}
}

func DeleteZone(log *slog.Logger, repo ZoneRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.access.DeleteZone"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteAccessZone(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("access zone not found", slog.Int("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Access zone not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("access zone has dependencies", slog.Int("id", id))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Cannot delete: access zone is in use"))
				return
			}
			log.Error("failed to delete access zone", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete access zone"))
			return
		}

		log.Info("access zone deleted", slog.Int("id", id))
		render.Status(r, http.StatusNoContent)
	}
}

// --- Card Handlers ---

func GetCards(log *slog.Logger, repo CardRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.access.GetCards"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.AccessCardFilter
		q := r.URL.Query()

		if empIDStr := q.Get("employee_id"); empIDStr != "" {
			val, err := strconv.ParseInt(empIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'employee_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'employee_id' parameter"))
				return
			}
			filter.EmployeeID = &val
		}

		if isActiveStr := q.Get("is_active"); isActiveStr != "" {
			val := isActiveStr == "true"
			filter.IsActive = &val
		}

		if cardType := q.Get("card_type"); cardType != "" {
			filter.CardType = &cardType
		}

		if search := q.Get("search"); search != "" {
			filter.Search = &search
		}

		if limitStr := q.Get("limit"); limitStr != "" {
			val, err := strconv.Atoi(limitStr)
			if err != nil {
				log.Warn("invalid 'limit' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'limit' parameter"))
				return
			}
			if val < 1 || val > 1000 {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Limit must be between 1 and 1000"))
				return
			}
			filter.Limit = val
		}

		if offsetStr := q.Get("offset"); offsetStr != "" {
			val, err := strconv.Atoi(offsetStr)
			if err != nil {
				log.Warn("invalid 'offset' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'offset' parameter"))
				return
			}
			if val < 0 {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Offset cannot be negative"))
				return
			}
			filter.Offset = val
		}

		cards, err := repo.GetAccessCards(r.Context(), filter)
		if err != nil {
			log.Error("failed to get access cards", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve access cards"))
			return
		}

		log.Info("successfully retrieved access cards", slog.Int("count", len(cards)))
		render.JSON(w, r, cards)
	}
}

func GetCardByID(log *slog.Logger, repo CardRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.access.GetCardByID"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		card, err := repo.GetAccessCardByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("access card not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Access card not found"))
				return
			}
			log.Error("failed to get access card", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve access card"))
			return
		}

		render.JSON(w, r, card)
	}
}

func AddCard(log *slog.Logger, repo CardRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.access.AddCard"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddAccessCardRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		id, err := repo.AddAccessCard(r.Context(), req)
		if err != nil {
			if errors.Is(err, storage.ErrUniqueViolation) {
				log.Warn("duplicate card number")
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Card number already exists"))
				return
			}
			log.Error("failed to add access card", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add access card"))
			return
		}

		log.Info("access card added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

func EditCard(log *slog.Logger, repo CardRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.access.EditCard"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.EditAccessCardRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.EditAccessCard(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("access card not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Access card not found"))
				return
			}
			if errors.Is(err, storage.ErrUniqueViolation) {
				log.Warn("duplicate card number")
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Card number already exists"))
				return
			}
			log.Error("failed to update access card", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update access card"))
			return
		}

		log.Info("access card updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

func DeleteCard(log *slog.Logger, repo CardRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.access.DeleteCard"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteAccessCard(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("access card not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Access card not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("access card has dependencies", slog.Int64("id", id))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Cannot delete: access card is in use"))
				return
			}
			log.Error("failed to delete access card", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete access card"))
			return
		}

		log.Info("access card deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}

// DeactivateRequest represents a card deactivation request
type DeactivateRequest struct {
	Reason string `json:"reason" validate:"required"`
}

func DeactivateCard(log *slog.Logger, repo CardRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.access.DeactivateCard"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req DeactivateRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		err = repo.DeactivateAccessCard(r.Context(), id, req.Reason)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("access card not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Access card not found"))
				return
			}
			log.Error("failed to deactivate access card", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to deactivate access card"))
			return
		}

		log.Info("access card deactivated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

// --- Card Zone Access Handlers ---

func GetCardZoneAccess(log *slog.Logger, repo CardZoneAccessRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.access.GetCardZoneAccess"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.CardZoneAccessFilter
		q := r.URL.Query()

		if cardIDStr := q.Get("card_id"); cardIDStr != "" {
			val, err := strconv.ParseInt(cardIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'card_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'card_id' parameter"))
				return
			}
			filter.CardID = &val
		}

		if zoneIDStr := q.Get("zone_id"); zoneIDStr != "" {
			val, err := strconv.Atoi(zoneIDStr)
			if err != nil {
				log.Warn("invalid 'zone_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'zone_id' parameter"))
				return
			}
			filter.ZoneID = &val
		}

		access, err := repo.GetCardZoneAccess(r.Context(), filter)
		if err != nil {
			log.Error("failed to get card zone access", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve card zone access"))
			return
		}

		log.Info("successfully retrieved card zone access", slog.Int("count", len(access)))
		render.JSON(w, r, access)
	}
}

func AddCardZoneAccess(log *slog.Logger, repo CardZoneAccessRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.access.AddCardZoneAccess"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddCardZoneAccessRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		// Get grantedBy from JWT claims
		var grantedBy *int64
		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if ok {
			grantedBy = &claims.UserID
		}

		id, err := repo.AddCardZoneAccess(r.Context(), req, grantedBy)
		if err != nil {
			if errors.Is(err, storage.ErrUniqueViolation) {
				log.Warn("duplicate card zone access")
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Card already has access to this zone"))
				return
			}
			log.Error("failed to add card zone access", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add card zone access"))
			return
		}

		log.Info("card zone access added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

func DeleteCardZoneAccess(log *slog.Logger, repo CardZoneAccessRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.access.DeleteCardZoneAccess"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteCardZoneAccess(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("card zone access not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Card zone access not found"))
				return
			}
			log.Error("failed to delete card zone access", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete card zone access"))
			return
		}

		log.Info("card zone access deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}

// --- Access Log Handlers ---

func GetAccessLogs(log *slog.Logger, repo AccessLogRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.access.GetAccessLogs"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.AccessLogFilter
		q := r.URL.Query()

		if cardIDStr := q.Get("card_id"); cardIDStr != "" {
			val, err := strconv.ParseInt(cardIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'card_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'card_id' parameter"))
				return
			}
			filter.CardID = &val
		}

		if empIDStr := q.Get("employee_id"); empIDStr != "" {
			val, err := strconv.ParseInt(empIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'employee_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'employee_id' parameter"))
				return
			}
			filter.EmployeeID = &val
		}

		if zoneIDStr := q.Get("zone_id"); zoneIDStr != "" {
			val, err := strconv.Atoi(zoneIDStr)
			if err != nil {
				log.Warn("invalid 'zone_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'zone_id' parameter"))
				return
			}
			filter.ZoneID = &val
		}

		if eventType := q.Get("event_type"); eventType != "" {
			filter.EventType = &eventType
		}

		if fromTimeStr := q.Get("from_time"); fromTimeStr != "" {
			val, err := time.Parse(time.RFC3339, fromTimeStr)
			if err != nil {
				log.Warn("invalid 'from_time' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'from_time' parameter, use RFC3339 format"))
				return
			}
			filter.FromTime = &val
		}

		if toTimeStr := q.Get("to_time"); toTimeStr != "" {
			val, err := time.Parse(time.RFC3339, toTimeStr)
			if err != nil {
				log.Warn("invalid 'to_time' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'to_time' parameter, use RFC3339 format"))
				return
			}
			filter.ToTime = &val
		}

		if limitStr := q.Get("limit"); limitStr != "" {
			val, err := strconv.Atoi(limitStr)
			if err != nil {
				log.Warn("invalid 'limit' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'limit' parameter"))
				return
			}
			if val < 1 || val > 1000 {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Limit must be between 1 and 1000"))
				return
			}
			filter.Limit = val
		}

		if offsetStr := q.Get("offset"); offsetStr != "" {
			val, err := strconv.Atoi(offsetStr)
			if err != nil {
				log.Warn("invalid 'offset' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'offset' parameter"))
				return
			}
			if val < 0 {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Offset cannot be negative"))
				return
			}
			filter.Offset = val
		}

		logs, err := repo.GetAccessLogs(r.Context(), filter)
		if err != nil {
			log.Error("failed to get access logs", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve access logs"))
			return
		}

		log.Info("successfully retrieved access logs", slog.Int("count", len(logs)))
		render.JSON(w, r, logs)
	}
}

func AddAccessLog(log *slog.Logger, repo AccessLogRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.access.AddAccessLog"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddAccessLogRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		id, err := repo.AddAccessLog(r.Context(), req)
		if err != nil {
			log.Error("failed to add access log", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add access log"))
			return
		}

		log.Info("access log added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}
