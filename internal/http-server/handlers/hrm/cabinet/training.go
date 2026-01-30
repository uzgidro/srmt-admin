package cabinet

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto/hrm"
	"srmt-admin/internal/lib/logger/sl"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
	"srmt-admin/internal/storage"
)

// TrainingRepository defines the interface for training operations
type TrainingRepository interface {
	GetEmployeeByUserID(ctx context.Context, userID int64) (*hrmmodel.Employee, error)
	GetParticipants(ctx context.Context, filter hrm.ParticipantFilter) ([]*hrmmodel.TrainingParticipant, error)
	GetTrainingByID(ctx context.Context, id int64) (*hrmmodel.Training, error)
	GetCertificates(ctx context.Context, filter hrm.CertificateFilter) ([]*hrmmodel.Certificate, error)
}

// GetMyTraining returns training information for the currently authenticated employee
func GetMyTraining(log *slog.Logger, repo TrainingRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.cabinet.GetMyTraining"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Get claims from JWT
		claims, ok := mwauth.ClaimsFromContext(r.Context())
		if !ok {
			log.Error("could not get user claims from context")
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Unauthorized"))
			return
		}

		// Find employee by user_id
		employee, err := repo.GetEmployeeByUserID(r.Context(), claims.UserID)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("employee not found", slog.Int64("user_id", claims.UserID))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Employee profile not found"))
				return
			}
			log.Error("failed to get employee", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve employee"))
			return
		}

		// Get training participations
		participants, err := repo.GetParticipants(r.Context(), hrm.ParticipantFilter{
			EmployeeID: &employee.ID,
		})
		if err != nil {
			log.Error("failed to get training participations", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve training information"))
			return
		}

		// Get certificates
		certificates, err := repo.GetCertificates(r.Context(), hrm.CertificateFilter{
			EmployeeID: &employee.ID,
		})
		if err != nil {
			log.Error("failed to get certificates", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve certificates"))
			return
		}

		// Build response
		response := hrm.MyTrainingResponse{
			Enrollments:  make([]hrm.MyTrainingEnrollment, 0, len(participants)),
			Certificates: make([]hrm.MyCertificate, 0, len(certificates)),
		}

		// Convert enrollments
		for _, p := range participants {
			enrollment := hrm.MyTrainingEnrollment{
				ID:                p.ID,
				TrainingID:        p.TrainingID,
				Status:            p.Status,
				AttendancePercent: p.AttendancePercent,
				Score:             p.Score,
				Passed:            p.Passed,
				CompletedAt:       p.CompletedAt,
			}

			// Fetch training details
			training, err := repo.GetTrainingByID(r.Context(), p.TrainingID)
			if err == nil && training != nil {
				enrollment.TrainingTitle = training.Title
				enrollment.TrainingType = training.TrainingType
				enrollment.StartDate = training.StartDate
				enrollment.EndDate = training.EndDate
				enrollment.Location = training.Location
				enrollment.IsMandatory = training.IsMandatory
			}

			response.Enrollments = append(response.Enrollments, enrollment)
		}

		// Convert certificates
		for _, c := range certificates {
			response.Certificates = append(response.Certificates, hrm.MyCertificate{
				ID:                c.ID,
				Name:              c.Name,
				Issuer:            c.Issuer,
				CertificateNumber: c.CertificateNumber,
				IssuedDate:        c.IssuedDate,
				ExpiryDate:        c.ExpiryDate,
				IsExpired:         c.IsExpired,
				IsVerified:        c.IsVerified,
			})
		}

		log.Info("training info retrieved", slog.Int64("employee_id", employee.ID),
			slog.Int("enrollments", len(participants)),
			slog.Int("certificates", len(certificates)))
		render.JSON(w, r, response)
	}
}
