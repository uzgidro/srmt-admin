package recruiting

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

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto/hrm"
	"srmt-admin/internal/lib/logger/sl"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
	"srmt-admin/internal/storage"
)

// VacancyRepository defines the interface for vacancy operations
type VacancyRepository interface {
	AddVacancy(ctx context.Context, req hrm.AddVacancyRequest) (int64, error)
	GetVacancyByID(ctx context.Context, id int64) (*hrmmodel.Vacancy, error)
	GetVacancies(ctx context.Context, filter hrm.VacancyFilter) ([]*hrmmodel.Vacancy, error)
	EditVacancy(ctx context.Context, id int64, req hrm.EditVacancyRequest) error
	PublishVacancy(ctx context.Context, id int64, publish bool) error
	CloseVacancy(ctx context.Context, id int64, status string) error
	DeleteVacancy(ctx context.Context, id int64) error
}

// CandidateRepository defines the interface for candidate operations
type CandidateRepository interface {
	AddCandidate(ctx context.Context, req hrm.AddCandidateRequest) (int64, error)
	GetCandidateByID(ctx context.Context, id int64) (*hrmmodel.Candidate, error)
	GetCandidates(ctx context.Context, filter hrm.CandidateFilter) ([]*hrmmodel.Candidate, error)
	EditCandidate(ctx context.Context, id int64, req hrm.EditCandidateRequest) error
	MoveCandidateStatus(ctx context.Context, id int64, req hrm.MoveCandidateRequest) error
	DeleteCandidate(ctx context.Context, id int64) error
}

// InterviewRepository defines the interface for interview operations
type InterviewRepository interface {
	AddInterview(ctx context.Context, req hrm.AddInterviewRequest) (int64, error)
	GetInterviewByID(ctx context.Context, id int64) (*hrmmodel.Interview, error)
	GetInterviews(ctx context.Context, filter hrm.InterviewFilter) ([]*hrmmodel.Interview, error)
	EditInterview(ctx context.Context, id int64, req hrm.EditInterviewRequest) error
	CompleteInterview(ctx context.Context, id int64, req hrm.CompleteInterviewRequest) error
	CancelInterview(ctx context.Context, id int64, reason string) error
	DeleteInterview(ctx context.Context, id int64) error
}

// IDResponse represents a response with ID
type IDResponse struct {
	resp.Response
	ID int64 `json:"id"`
}

// --- Vacancy Handlers ---

// GetVacancies returns all vacancies with filters
func GetVacancies(log *slog.Logger, repo VacancyRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.GetVacancies"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.VacancyFilter
		q := r.URL.Query()

		if deptIDStr := q.Get("department_id"); deptIDStr != "" {
			val, err := strconv.ParseInt(deptIDStr, 10, 64)
			if err == nil {
				filter.DepartmentID = &val
			}
		}

		if posIDStr := q.Get("position_id"); posIDStr != "" {
			val, err := strconv.ParseInt(posIDStr, 10, 64)
			if err == nil {
				filter.PositionID = &val
			}
		}

		if status := q.Get("status"); status != "" {
			filter.Status = &status
		}

		if empType := q.Get("employment_type"); empType != "" {
			filter.EmploymentType = &empType
		}

		if limitStr := q.Get("limit"); limitStr != "" {
			val, err := strconv.Atoi(limitStr)
			if err == nil {
				filter.Limit = val
			}
		}

		if offsetStr := q.Get("offset"); offsetStr != "" {
			val, err := strconv.Atoi(offsetStr)
			if err == nil {
				filter.Offset = val
			}
		}

		vacancies, err := repo.GetVacancies(r.Context(), filter)
		if err != nil {
			log.Error("failed to get vacancies", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve vacancies"))
			return
		}

		log.Info("successfully retrieved vacancies", slog.Int("count", len(vacancies)))
		render.JSON(w, r, vacancies)
	}
}

// GetVacancyByID returns a vacancy by ID
func GetVacancyByID(log *slog.Logger, repo VacancyRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.GetVacancyByID"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		vacancy, err := repo.GetVacancyByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("vacancy not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Vacancy not found"))
				return
			}
			log.Error("failed to get vacancy", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve vacancy"))
			return
		}

		log.Info("successfully retrieved vacancy", slog.Int64("id", vacancy.ID))
		render.JSON(w, r, vacancy)
	}
}

// AddVacancy creates a new vacancy
func AddVacancy(log *slog.Logger, repo VacancyRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.AddVacancy"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddVacancyRequest
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

		id, err := repo.AddVacancy(r.Context(), req)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid department_id or position_id"))
				return
			}
			log.Error("failed to add vacancy", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add vacancy"))
			return
		}

		log.Info("vacancy added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

// EditVacancy updates a vacancy
func EditVacancy(log *slog.Logger, repo VacancyRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.EditVacancy"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.EditVacancyRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.EditVacancy(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("vacancy not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Vacancy not found"))
				return
			}
			log.Error("failed to update vacancy", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update vacancy"))
			return
		}

		log.Info("vacancy updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

// PublishVacancy publishes a vacancy
func PublishVacancy(log *slog.Logger, repo VacancyRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.PublishVacancy"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.PublishVacancy(r.Context(), id, true)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("vacancy not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Vacancy not found"))
				return
			}
			log.Error("failed to publish vacancy", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to publish vacancy"))
			return
		}

		log.Info("vacancy published", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

// CloseVacancy closes a vacancy
func CloseVacancy(log *slog.Logger, repo VacancyRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.CloseVacancy"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.CloseVacancy(r.Context(), id, "closed")
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("vacancy not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Vacancy not found"))
				return
			}
			log.Error("failed to close vacancy", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to close vacancy"))
			return
		}

		log.Info("vacancy closed", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

// DeleteVacancy deletes a vacancy
func DeleteVacancy(log *slog.Logger, repo VacancyRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.DeleteVacancy"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteVacancy(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("vacancy not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Vacancy not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("vacancy has dependencies", slog.Int64("id", id))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Cannot delete: vacancy has candidates"))
				return
			}
			log.Error("failed to delete vacancy", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete vacancy"))
			return
		}

		log.Info("vacancy deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}

// --- Candidate Handlers ---

// GetCandidates returns all candidates with filters
func GetCandidates(log *slog.Logger, repo CandidateRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.GetCandidates"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.CandidateFilter
		q := r.URL.Query()

		if vacancyIDStr := q.Get("vacancy_id"); vacancyIDStr != "" {
			val, err := strconv.ParseInt(vacancyIDStr, 10, 64)
			if err == nil {
				filter.VacancyID = &val
			}
		}

		if status := q.Get("status"); status != "" {
			filter.Status = &status
		}

		if search := q.Get("search"); search != "" {
			filter.Search = &search
		}

		if limitStr := q.Get("limit"); limitStr != "" {
			val, err := strconv.Atoi(limitStr)
			if err == nil {
				filter.Limit = val
			}
		}

		if offsetStr := q.Get("offset"); offsetStr != "" {
			val, err := strconv.Atoi(offsetStr)
			if err == nil {
				filter.Offset = val
			}
		}

		candidates, err := repo.GetCandidates(r.Context(), filter)
		if err != nil {
			log.Error("failed to get candidates", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve candidates"))
			return
		}

		log.Info("successfully retrieved candidates", slog.Int("count", len(candidates)))
		render.JSON(w, r, candidates)
	}
}

// GetCandidateByID returns a candidate by ID
func GetCandidateByID(log *slog.Logger, repo CandidateRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.GetCandidateByID"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		candidate, err := repo.GetCandidateByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("candidate not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Candidate not found"))
				return
			}
			log.Error("failed to get candidate", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve candidate"))
			return
		}

		log.Info("successfully retrieved candidate", slog.Int64("id", candidate.ID))
		render.JSON(w, r, candidate)
	}
}

// AddCandidate creates a new candidate
func AddCandidate(log *slog.Logger, repo CandidateRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.AddCandidate"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddCandidateRequest
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

		id, err := repo.AddCandidate(r.Context(), req)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid vacancy_id"))
				return
			}
			log.Error("failed to add candidate", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add candidate"))
			return
		}

		log.Info("candidate added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

// EditCandidate updates a candidate
func EditCandidate(log *slog.Logger, repo CandidateRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.EditCandidate"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.EditCandidateRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.EditCandidate(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("candidate not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Candidate not found"))
				return
			}
			log.Error("failed to update candidate", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update candidate"))
			return
		}

		log.Info("candidate updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

// MoveCandidateStatus moves candidate to a new status
func MoveCandidateStatus(log *slog.Logger, repo CandidateRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.MoveCandidateStatus"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.MoveCandidateRequest
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

		err = repo.MoveCandidateStatus(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("candidate not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Candidate not found"))
				return
			}
			log.Error("failed to move candidate status", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to move candidate status"))
			return
		}

		log.Info("candidate status moved", slog.Int64("id", id), slog.String("status", req.Status))
		render.JSON(w, r, resp.OK())
	}
}

// DeleteCandidate deletes a candidate
func DeleteCandidate(log *slog.Logger, repo CandidateRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.DeleteCandidate"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteCandidate(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("candidate not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Candidate not found"))
				return
			}
			log.Error("failed to delete candidate", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete candidate"))
			return
		}

		log.Info("candidate deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}

// --- Interview Handlers ---

// GetInterviews returns all interviews with filters
func GetInterviews(log *slog.Logger, repo InterviewRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.GetInterviews"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.InterviewFilter
		q := r.URL.Query()

		if candidateIDStr := q.Get("candidate_id"); candidateIDStr != "" {
			val, err := strconv.ParseInt(candidateIDStr, 10, 64)
			if err == nil {
				filter.CandidateID = &val
			}
		}

		if interviewerIDStr := q.Get("interviewer_id"); interviewerIDStr != "" {
			val, err := strconv.ParseInt(interviewerIDStr, 10, 64)
			if err == nil {
				filter.InterviewerID = &val
			}
		}

		if status := q.Get("status"); status != "" {
			filter.Status = &status
		}

		if fromDateStr := q.Get("from_date"); fromDateStr != "" {
			val, err := time.Parse(time.DateOnly, fromDateStr)
			if err == nil {
				filter.FromDate = &val
			}
		}

		if toDateStr := q.Get("to_date"); toDateStr != "" {
			val, err := time.Parse(time.DateOnly, toDateStr)
			if err == nil {
				filter.ToDate = &val
			}
		}

		interviews, err := repo.GetInterviews(r.Context(), filter)
		if err != nil {
			log.Error("failed to get interviews", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve interviews"))
			return
		}

		log.Info("successfully retrieved interviews", slog.Int("count", len(interviews)))
		render.JSON(w, r, interviews)
	}
}

// GetInterviewByID returns an interview by ID
func GetInterviewByID(log *slog.Logger, repo InterviewRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.GetInterviewByID"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		interview, err := repo.GetInterviewByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("interview not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Interview not found"))
				return
			}
			log.Error("failed to get interview", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve interview"))
			return
		}

		log.Info("successfully retrieved interview", slog.Int64("id", interview.ID))
		render.JSON(w, r, interview)
	}
}

// AddInterview creates a new interview
func AddInterview(log *slog.Logger, repo InterviewRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.AddInterview"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddInterviewRequest
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

		id, err := repo.AddInterview(r.Context(), req)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid candidate_id or interviewer_id"))
				return
			}
			log.Error("failed to add interview", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add interview"))
			return
		}

		log.Info("interview added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

// EditInterview updates an interview
func EditInterview(log *slog.Logger, repo InterviewRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.EditInterview"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.EditInterviewRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.EditInterview(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("interview not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Interview not found"))
				return
			}
			log.Error("failed to update interview", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update interview"))
			return
		}

		log.Info("interview updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

// CompleteInterview completes an interview with result
func CompleteInterview(log *slog.Logger, repo InterviewRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.CompleteInterview"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.CompleteInterviewRequest
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

		err = repo.CompleteInterview(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("interview not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Interview not found"))
				return
			}
			log.Error("failed to complete interview", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to complete interview"))
			return
		}

		log.Info("interview completed", slog.Int64("id", id), slog.String("recommendation", req.Recommendation))
		render.JSON(w, r, resp.OK())
	}
}

// CancelInterview cancels an interview
func CancelInterview(log *slog.Logger, repo InterviewRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.CancelInterview"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.CancelInterviewRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.CancelInterview(r.Context(), id, req.Reason)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("interview not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Interview not found"))
				return
			}
			log.Error("failed to cancel interview", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to cancel interview"))
			return
		}

		log.Info("interview cancelled", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

// DeleteInterview deletes an interview
func DeleteInterview(log *slog.Logger, repo InterviewRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.recruiting.DeleteInterview"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteInterview(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("interview not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Interview not found"))
				return
			}
			log.Error("failed to delete interview", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete interview"))
			return
		}

		log.Info("interview deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}
