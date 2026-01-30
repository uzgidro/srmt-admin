package performance

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

type ReviewRepository interface {
	AddPerformanceReview(ctx context.Context, req hrm.AddPerformanceReviewRequest) (int64, error)
	GetPerformanceReviewByID(ctx context.Context, id int64) (*hrmmodel.PerformanceReview, error)
	GetPerformanceReviews(ctx context.Context, filter hrm.PerformanceReviewFilter) ([]*hrmmodel.PerformanceReview, error)
	EditPerformanceReview(ctx context.Context, id int64, req hrm.EditPerformanceReviewRequest) error
	SubmitSelfReview(ctx context.Context, id int64, req hrm.SubmitSelfReviewRequest) error
	SubmitManagerReview(ctx context.Context, id int64, req hrm.SubmitManagerReviewRequest) error
	CalibrateReview(ctx context.Context, id int64, calibratorID int64, req hrm.CalibrateReviewRequest) error
	DeletePerformanceReview(ctx context.Context, id int64) error
}

type GoalRepository interface {
	AddPerformanceGoal(ctx context.Context, req hrm.AddPerformanceGoalRequest) (int64, error)
	GetPerformanceGoalByID(ctx context.Context, id int64) (*hrmmodel.PerformanceGoal, error)
	GetPerformanceGoals(ctx context.Context, filter hrm.PerformanceGoalFilter) ([]*hrmmodel.PerformanceGoal, error)
	EditPerformanceGoal(ctx context.Context, id int64, req hrm.EditPerformanceGoalRequest) error
	UpdateGoalProgress(ctx context.Context, id int64, req hrm.UpdateGoalProgressRequest) error
	RateGoal(ctx context.Context, id int64, req hrm.RateGoalRequest) error
	DeletePerformanceGoal(ctx context.Context, id int64) error
}

type KPIRepository interface {
	AddKPI(ctx context.Context, req hrm.AddKPIRequest) (int64, error)
	GetKPIByID(ctx context.Context, id int64) (*hrmmodel.KPI, error)
	GetKPIs(ctx context.Context, filter hrm.KPIFilter) ([]*hrmmodel.KPI, error)
	EditKPI(ctx context.Context, id int64, req hrm.EditKPIRequest) error
	UpdateKPIValue(ctx context.Context, id int64, req hrm.UpdateKPIValueRequest) error
	RateKPI(ctx context.Context, id int64, req hrm.RateKPIRequest) error
	DeleteKPI(ctx context.Context, id int64) error
}

// IDResponse represents a response with ID
type IDResponse struct {
	resp.Response
	ID int64 `json:"id"`
}

// --- Review Handlers ---

func GetReviews(log *slog.Logger, repo ReviewRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.GetReviews"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.PerformanceReviewFilter
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

		if reviewerIDStr := q.Get("reviewer_id"); reviewerIDStr != "" {
			val, err := strconv.ParseInt(reviewerIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'reviewer_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'reviewer_id' parameter"))
				return
			}
			filter.ReviewerID = &val
		}

		if reviewType := q.Get("review_type"); reviewType != "" {
			filter.ReviewType = &reviewType
		}

		if status := q.Get("status"); status != "" {
			filter.Status = &status
		}

		if fromDateStr := q.Get("from_date"); fromDateStr != "" {
			val, err := time.Parse(time.DateOnly, fromDateStr)
			if err != nil {
				log.Warn("invalid 'from_date' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'from_date' parameter, use YYYY-MM-DD"))
				return
			}
			filter.FromDate = &val
		}

		if toDateStr := q.Get("to_date"); toDateStr != "" {
			val, err := time.Parse(time.DateOnly, toDateStr)
			if err != nil {
				log.Warn("invalid 'to_date' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'to_date' parameter, use YYYY-MM-DD"))
				return
			}
			filter.ToDate = &val
		}

		if limitStr := q.Get("limit"); limitStr != "" {
			val, _ := strconv.Atoi(limitStr)
			filter.Limit = val
		}

		if offsetStr := q.Get("offset"); offsetStr != "" {
			val, _ := strconv.Atoi(offsetStr)
			filter.Offset = val
		}

		reviews, err := repo.GetPerformanceReviews(r.Context(), filter)
		if err != nil {
			log.Error("failed to get reviews", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve reviews"))
			return
		}

		log.Info("successfully retrieved reviews", slog.Int("count", len(reviews)))
		render.JSON(w, r, reviews)
	}
}

func GetReviewByID(log *slog.Logger, repo ReviewRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.GetReviewByID"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		review, err := repo.GetPerformanceReviewByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("review not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Review not found"))
				return
			}
			log.Error("failed to get review", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve review"))
			return
		}

		render.JSON(w, r, review)
	}
}

func AddReview(log *slog.Logger, repo ReviewRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.AddReview"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddPerformanceReviewRequest
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

		id, err := repo.AddPerformanceReview(r.Context(), req)
		if err != nil {
			log.Error("failed to add review", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add review"))
			return
		}

		log.Info("review added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

func EditReview(log *slog.Logger, repo ReviewRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.EditReview"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.EditPerformanceReviewRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.EditPerformanceReview(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("review not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Review not found"))
				return
			}
			log.Error("failed to update review", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update review"))
			return
		}

		log.Info("review updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

func SubmitSelfReview(log *slog.Logger, repo ReviewRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.SubmitSelfReview"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.SubmitSelfReviewRequest
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

		err = repo.SubmitSelfReview(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("review not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Review not found"))
				return
			}
			log.Error("failed to submit self-review", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to submit self-review"))
			return
		}

		log.Info("self-review submitted", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

func SubmitManagerReview(log *slog.Logger, repo ReviewRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.SubmitManagerReview"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.SubmitManagerReviewRequest
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

		err = repo.SubmitManagerReview(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("review not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Review not found"))
				return
			}
			log.Error("failed to submit manager review", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to submit manager review"))
			return
		}

		log.Info("manager review submitted", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

func CalibrateReview(log *slog.Logger, repo ReviewRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.CalibrateReview"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.CalibrateReviewRequest
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

		// Get calibrator ID from JWT claims
		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			log.Error("failed to get claims from context")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Unauthorized"))
			return
		}
		calibratorID := claims.UserID

		err = repo.CalibrateReview(r.Context(), id, calibratorID, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("review not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Review not found"))
				return
			}
			log.Error("failed to calibrate review", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to calibrate review"))
			return
		}

		log.Info("review calibrated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

func DeleteReview(log *slog.Logger, repo ReviewRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.DeleteReview"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeletePerformanceReview(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("review not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Review not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("review has dependencies", slog.Int64("id", id))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Cannot delete: review has associated goals"))
				return
			}
			log.Error("failed to delete review", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete review"))
			return
		}

		log.Info("review deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}

// --- Goal Handlers ---

func GetGoals(log *slog.Logger, repo GoalRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.GetGoals"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.PerformanceGoalFilter
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

		if reviewIDStr := q.Get("review_id"); reviewIDStr != "" {
			val, err := strconv.ParseInt(reviewIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'review_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'review_id' parameter"))
				return
			}
			filter.ReviewID = &val
		}

		if status := q.Get("status"); status != "" {
			filter.Status = &status
		}

		if category := q.Get("category"); category != "" {
			filter.Category = &category
		}

		if limitStr := q.Get("limit"); limitStr != "" {
			val, _ := strconv.Atoi(limitStr)
			filter.Limit = val
		}

		if offsetStr := q.Get("offset"); offsetStr != "" {
			val, _ := strconv.Atoi(offsetStr)
			filter.Offset = val
		}

		goals, err := repo.GetPerformanceGoals(r.Context(), filter)
		if err != nil {
			log.Error("failed to get goals", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve goals"))
			return
		}

		log.Info("successfully retrieved goals", slog.Int("count", len(goals)))
		render.JSON(w, r, goals)
	}
}

func GetGoalByID(log *slog.Logger, repo GoalRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.GetGoalByID"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		goal, err := repo.GetPerformanceGoalByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("goal not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Goal not found"))
				return
			}
			log.Error("failed to get goal", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve goal"))
			return
		}

		render.JSON(w, r, goal)
	}
}

func AddGoal(log *slog.Logger, repo GoalRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.AddGoal"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddPerformanceGoalRequest
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

		id, err := repo.AddPerformanceGoal(r.Context(), req)
		if err != nil {
			log.Error("failed to add goal", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add goal"))
			return
		}

		log.Info("goal added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

func EditGoal(log *slog.Logger, repo GoalRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.EditGoal"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.EditPerformanceGoalRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.EditPerformanceGoal(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("goal not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Goal not found"))
				return
			}
			log.Error("failed to update goal", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update goal"))
			return
		}

		log.Info("goal updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

func UpdateGoalProgress(log *slog.Logger, repo GoalRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.UpdateGoalProgress"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.UpdateGoalProgressRequest
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

		err = repo.UpdateGoalProgress(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("goal not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Goal not found"))
				return
			}
			log.Error("failed to update goal progress", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update goal progress"))
			return
		}

		log.Info("goal progress updated", slog.Int64("id", id), slog.Int("progress", req.Progress))
		render.JSON(w, r, resp.OK())
	}
}

func RateGoal(log *slog.Logger, repo GoalRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.RateGoal"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.RateGoalRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.RateGoal(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("goal not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Goal not found"))
				return
			}
			log.Error("failed to rate goal", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to rate goal"))
			return
		}

		log.Info("goal rated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

func DeleteGoal(log *slog.Logger, repo GoalRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.DeleteGoal"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeletePerformanceGoal(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("goal not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Goal not found"))
				return
			}
			log.Error("failed to delete goal", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete goal"))
			return
		}

		log.Info("goal deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}

// --- KPI Handlers ---

func GetKPIs(log *slog.Logger, repo KPIRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.GetKPIs"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.KPIFilter
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

		if yearStr := q.Get("year"); yearStr != "" {
			val, err := strconv.Atoi(yearStr)
			if err != nil {
				log.Warn("invalid 'year' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'year' parameter"))
				return
			}
			filter.Year = &val
		}

		if monthStr := q.Get("month"); monthStr != "" {
			val, err := strconv.Atoi(monthStr)
			if err != nil {
				log.Warn("invalid 'month' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'month' parameter"))
				return
			}
			filter.Month = &val
		}

		if quarterStr := q.Get("quarter"); quarterStr != "" {
			val, err := strconv.Atoi(quarterStr)
			if err != nil {
				log.Warn("invalid 'quarter' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'quarter' parameter"))
				return
			}
			filter.Quarter = &val
		}

		if category := q.Get("category"); category != "" {
			filter.Category = &category
		}

		if limitStr := q.Get("limit"); limitStr != "" {
			val, _ := strconv.Atoi(limitStr)
			filter.Limit = val
		}

		if offsetStr := q.Get("offset"); offsetStr != "" {
			val, _ := strconv.Atoi(offsetStr)
			filter.Offset = val
		}

		kpis, err := repo.GetKPIs(r.Context(), filter)
		if err != nil {
			log.Error("failed to get KPIs", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve KPIs"))
			return
		}

		log.Info("successfully retrieved KPIs", slog.Int("count", len(kpis)))
		render.JSON(w, r, kpis)
	}
}

func GetKPIByID(log *slog.Logger, repo KPIRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.GetKPIByID"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		kpi, err := repo.GetKPIByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("KPI not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("KPI not found"))
				return
			}
			log.Error("failed to get KPI", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve KPI"))
			return
		}

		render.JSON(w, r, kpi)
	}
}

func AddKPI(log *slog.Logger, repo KPIRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.AddKPI"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddKPIRequest
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

		id, err := repo.AddKPI(r.Context(), req)
		if err != nil {
			log.Error("failed to add KPI", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add KPI"))
			return
		}

		log.Info("KPI added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

func EditKPI(log *slog.Logger, repo KPIRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.EditKPI"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.EditKPIRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.EditKPI(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("KPI not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("KPI not found"))
				return
			}
			log.Error("failed to update KPI", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update KPI"))
			return
		}

		log.Info("KPI updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

func UpdateKPIValue(log *slog.Logger, repo KPIRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.UpdateKPIValue"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.UpdateKPIValueRequest
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

		err = repo.UpdateKPIValue(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("KPI not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("KPI not found"))
				return
			}
			log.Error("failed to update KPI value", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update KPI value"))
			return
		}

		log.Info("KPI value updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

func RateKPI(log *slog.Logger, repo KPIRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.RateKPI"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.RateKPIRequest
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

		err = repo.RateKPI(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("KPI not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("KPI not found"))
				return
			}
			log.Error("failed to rate KPI", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to rate KPI"))
			return
		}

		log.Info("KPI rated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

func DeleteKPI(log *slog.Logger, repo KPIRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.performance.DeleteKPI"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteKPI(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("KPI not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("KPI not found"))
				return
			}
			log.Error("failed to delete KPI", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete KPI"))
			return
		}

		log.Info("KPI deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}
