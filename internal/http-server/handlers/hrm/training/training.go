package training

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

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

// TrainingRepository defines the interface for training operations
type TrainingRepository interface {
	AddTraining(ctx context.Context, req hrm.AddTrainingRequest) (int64, error)
	GetTrainingByID(ctx context.Context, id int64) (*hrmmodel.Training, error)
	GetTrainings(ctx context.Context, filter hrm.TrainingFilter) ([]*hrmmodel.Training, error)
	EditTraining(ctx context.Context, id int64, req hrm.EditTrainingRequest) error
	DeleteTraining(ctx context.Context, id int64) error
}

// ParticipantRepository defines the interface for participant operations
type ParticipantRepository interface {
	EnrollParticipant(ctx context.Context, req hrm.EnrollParticipantRequest, enrolledBy *int64) (int64, error)
	GetParticipants(ctx context.Context, filter hrm.ParticipantFilter) ([]*hrmmodel.TrainingParticipant, error)
	CompleteParticipantTraining(ctx context.Context, id int64, req hrm.CompleteTrainingRequest) error
}

// CertificateRepository defines the interface for certificate operations
type CertificateRepository interface {
	AddCertificate(ctx context.Context, req hrm.AddCertificateRequest) (int64, error)
	GetCertificates(ctx context.Context, filter hrm.CertificateFilter) ([]*hrmmodel.Certificate, error)
	EditCertificate(ctx context.Context, id int64, req hrm.EditCertificateRequest) error
	DeleteCertificate(ctx context.Context, id int64) error
}

// IDResponse represents a response with ID
type IDResponse struct {
	resp.Response
	ID int64 `json:"id"`
}

// --- Training Handlers ---

// GetTrainings returns trainings with filters
func GetTrainings(log *slog.Logger, repo TrainingRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.training.GetTrainings"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.TrainingFilter
		q := r.URL.Query()

		if category := q.Get("category"); category != "" {
			filter.Category = &category
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

		trainings, err := repo.GetTrainings(r.Context(), filter)
		if err != nil {
			log.Error("failed to get trainings", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve trainings"))
			return
		}

		log.Info("successfully retrieved trainings", slog.Int("count", len(trainings)))
		render.JSON(w, r, trainings)
	}
}

// GetTrainingByID returns a training by ID
func GetTrainingByID(log *slog.Logger, repo TrainingRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.training.GetTrainingByID"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		training, err := repo.GetTrainingByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("training not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Training not found"))
				return
			}
			log.Error("failed to get training", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve training"))
			return
		}

		log.Info("successfully retrieved training", slog.Int64("id", training.ID))
		render.JSON(w, r, training)
	}
}

// AddTraining creates a new training
func AddTraining(log *slog.Logger, repo TrainingRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.training.AddTraining"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddTrainingRequest
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

		id, err := repo.AddTraining(r.Context(), req)
		if err != nil {
			log.Error("failed to add training", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add training"))
			return
		}

		log.Info("training added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

// EditTraining updates a training
func EditTraining(log *slog.Logger, repo TrainingRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.training.EditTraining"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.EditTrainingRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.EditTraining(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("training not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Training not found"))
				return
			}
			log.Error("failed to update training", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update training"))
			return
		}

		log.Info("training updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

// DeleteTraining deletes a training
func DeleteTraining(log *slog.Logger, repo TrainingRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.training.DeleteTraining"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteTraining(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("training not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Training not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("training has dependencies", slog.Int64("id", id))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Cannot delete: training has participants"))
				return
			}
			log.Error("failed to delete training", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete training"))
			return
		}

		log.Info("training deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}

// --- Participant Handlers ---

// GetParticipants returns training participants with filters
func GetParticipants(log *slog.Logger, repo ParticipantRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.training.GetParticipants"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.ParticipantFilter
		q := r.URL.Query()

		if trainingIDStr := q.Get("training_id"); trainingIDStr != "" {
			val, err := strconv.ParseInt(trainingIDStr, 10, 64)
			if err == nil {
				filter.TrainingID = &val
			}
		}

		if empIDStr := q.Get("employee_id"); empIDStr != "" {
			val, err := strconv.ParseInt(empIDStr, 10, 64)
			if err == nil {
				filter.EmployeeID = &val
			}
		}

		if status := q.Get("status"); status != "" {
			filter.Status = &status
		}

		participants, err := repo.GetParticipants(r.Context(), filter)
		if err != nil {
			log.Error("failed to get participants", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve participants"))
			return
		}

		log.Info("successfully retrieved participants", slog.Int("count", len(participants)))
		render.JSON(w, r, participants)
	}
}

// EnrollParticipant enrolls an employee in a training
func EnrollParticipant(log *slog.Logger, repo ParticipantRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.training.EnrollParticipant"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.EnrollParticipantRequest
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

		id, err := repo.EnrollParticipant(r.Context(), req, nil)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid training_id or employee_id"))
				return
			}
			if errors.Is(err, storage.ErrUniqueViolation) {
				log.Warn("duplicate enrollment")
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Employee is already enrolled in this training"))
				return
			}
			log.Error("failed to enroll participant", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to enroll participant"))
			return
		}

		log.Info("participant enrolled", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

// CompleteParticipantTraining marks training as completed for a participant
func CompleteParticipantTraining(log *slog.Logger, repo ParticipantRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.training.CompleteParticipantTraining"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.CompleteTrainingRequest
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

		err = repo.CompleteParticipantTraining(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("participant not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Participant not found"))
				return
			}
			log.Error("failed to complete training", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to complete training"))
			return
		}

		log.Info("training completed for participant", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

// --- Certificate Handlers ---

// GetCertificates returns certificates with filters
func GetCertificates(log *slog.Logger, repo CertificateRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.training.GetCertificates"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.CertificateFilter
		q := r.URL.Query()

		if empIDStr := q.Get("employee_id"); empIDStr != "" {
			val, err := strconv.ParseInt(empIDStr, 10, 64)
			if err == nil {
				filter.EmployeeID = &val
			}
		}

		if search := q.Get("search"); search != "" {
			filter.Search = &search
		}

		if expiringDaysStr := q.Get("expiring_days"); expiringDaysStr != "" {
			val, err := strconv.Atoi(expiringDaysStr)
			if err == nil {
				filter.ExpiringDays = &val
			}
		}

		certificates, err := repo.GetCertificates(r.Context(), filter)
		if err != nil {
			log.Error("failed to get certificates", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve certificates"))
			return
		}

		log.Info("successfully retrieved certificates", slog.Int("count", len(certificates)))
		render.JSON(w, r, certificates)
	}
}

// AddCertificate creates a new certificate
func AddCertificate(log *slog.Logger, repo CertificateRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.training.AddCertificate"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddCertificateRequest
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

		id, err := repo.AddCertificate(r.Context(), req)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation", "employee_id", req.EmployeeID)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid employee_id"))
				return
			}
			log.Error("failed to add certificate", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add certificate"))
			return
		}

		log.Info("certificate added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

// EditCertificate updates a certificate
func EditCertificate(log *slog.Logger, repo CertificateRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.training.EditCertificate"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.EditCertificateRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.EditCertificate(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("certificate not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Certificate not found"))
				return
			}
			log.Error("failed to update certificate", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update certificate"))
			return
		}

		log.Info("certificate updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

// DeleteCertificate deletes a certificate
func DeleteCertificate(log *slog.Logger, repo CertificateRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.training.DeleteCertificate"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteCertificate(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("certificate not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Certificate not found"))
				return
			}
			log.Error("failed to delete certificate", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete certificate"))
			return
		}

		log.Info("certificate deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}
