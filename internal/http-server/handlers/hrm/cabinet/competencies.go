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

// CompetencyRepository defines the interface for competency operations
type CompetencyRepository interface {
	GetEmployeeByUserID(ctx context.Context, userID int64) (*hrmmodel.Employee, error)
	GetCompetencyAssessments(ctx context.Context, filter hrm.AssessmentFilter) ([]*hrmmodel.CompetencyAssessment, error)
	GetCompetencyScores(ctx context.Context, filter hrm.ScoreFilter) ([]*hrmmodel.CompetencyScore, error)
}

// GetMyCompetencies returns competencies for the currently authenticated employee
func GetMyCompetencies(log *slog.Logger, repo CompetencyRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.cabinet.GetMyCompetencies"
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

		// Get assessments for this employee
		assessments, err := repo.GetCompetencyAssessments(r.Context(), hrm.AssessmentFilter{
			EmployeeID: &employee.ID,
		})
		if err != nil {
			log.Error("failed to get assessments", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve assessments"))
			return
		}

		// Build response
		response := hrm.MyCompetenciesResponse{
			Scores:      make([]hrm.MyCompetencyScore, 0),
			Assessments: make([]hrm.MyAssessment, 0, len(assessments)),
		}

		// Get scores from the latest completed assessment
		for _, a := range assessments {
			response.Assessments = append(response.Assessments, hrm.MyAssessment{
				ID:             a.ID,
				AssessmentType: a.AssessmentType,
				Status:         a.Status,
				ScheduledDate:  a.StartedAt,
				CompletedAt:    a.CompletedAt,
			})

			// Get scores from completed assessments
			if a.Status == hrmmodel.AssessmentStatusCompleted {
				scores, err := repo.GetCompetencyScores(r.Context(), hrm.ScoreFilter{
					AssessmentID: &a.ID,
				})
				if err == nil {
					for _, s := range scores {
						score := hrm.MyCompetencyScore{
							CompetencyID:   int64(s.CompetencyID),
							Score:          s.Score,
							MaxScore:       5,
							AssessmentDate: a.AssessmentPeriodEnd,
						}

						if s.Competency != nil {
							score.CompetencyName = s.Competency.Name
							if s.Competency.Category != nil {
								score.CategoryName = s.Competency.Category.Name
							}
						}

						response.Scores = append(response.Scores, score)
					}
				}
			}
		}

		log.Info("competencies retrieved", slog.Int64("employee_id", employee.ID),
			slog.Int("scores", len(response.Scores)),
			slog.Int("assessments", len(assessments)))
		render.JSON(w, r, response)
	}
}
