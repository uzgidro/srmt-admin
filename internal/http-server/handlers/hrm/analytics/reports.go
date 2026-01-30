package analytics

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto/hrm"
	"srmt-admin/internal/lib/logger/sl"
)

// ReportsRepository defines the interface for reports operations
type ReportsRepository interface {
	GetHeadcountStats(ctx context.Context, filter hrm.AnalyticsFilter) (*hrm.HeadcountReportResponse, error)
	GetHeadcountTrend(ctx context.Context, filter hrm.AnalyticsFilter) (*hrm.HeadcountTrendResponse, error)
	GetTurnoverStats(ctx context.Context, filter hrm.AnalyticsFilter) (*hrm.TurnoverReportResponse, error)
	GetTurnoverTrend(ctx context.Context, filter hrm.AnalyticsFilter) (*hrm.TurnoverTrendResponse, error)
	GetAttendanceStats(ctx context.Context, filter hrm.AnalyticsFilter) (*hrm.AttendanceReportResponse, error)
	GetSalaryStats(ctx context.Context, filter hrm.AnalyticsFilter) (*hrm.SalaryReportResponse, error)
	GetSalaryTrend(ctx context.Context, filter hrm.AnalyticsFilter) (*hrm.SalaryTrendResponse, error)
	GetPerformanceReport(ctx context.Context, filter hrm.AnalyticsFilter) (*hrm.PerformanceReportResponse, error)
	GetTrainingReport(ctx context.Context, filter hrm.AnalyticsFilter) (*hrm.TrainingReportResponse, error)
	GetDemographicsStats(ctx context.Context) (*hrm.DemographicsReportResponse, error)
}

// parseAnalyticsFilter parses filter from query params
func parseAnalyticsFilter(r *http.Request) hrm.AnalyticsFilter {
	filter := hrm.AnalyticsFilter{}

	if orgIDStr := r.URL.Query().Get("organization_id"); orgIDStr != "" {
		if orgID, err := strconv.ParseInt(orgIDStr, 10, 64); err == nil {
			filter.OrganizationID = &orgID
		}
	}

	if deptIDStr := r.URL.Query().Get("department_id"); deptIDStr != "" {
		if deptID, err := strconv.ParseInt(deptIDStr, 10, 64); err == nil {
			filter.DepartmentID = &deptID
		}
	}

	return filter
}

// GetHeadcountReport returns headcount report
func GetHeadcountReport(log *slog.Logger, repo ReportsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.analytics.GetHeadcountReport"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		filter := parseAnalyticsFilter(r)

		report, err := repo.GetHeadcountStats(r.Context(), filter)
		if err != nil {
			log.Error("failed to get headcount report", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve headcount report"))
			return
		}

		log.Info("headcount report retrieved")
		render.JSON(w, r, report)
	}
}

// GetHeadcountTrend returns headcount trend
func GetHeadcountTrend(log *slog.Logger, repo ReportsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.analytics.GetHeadcountTrend"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		filter := parseAnalyticsFilter(r)

		trend, err := repo.GetHeadcountTrend(r.Context(), filter)
		if err != nil {
			log.Error("failed to get headcount trend", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve headcount trend"))
			return
		}

		log.Info("headcount trend retrieved")
		render.JSON(w, r, trend)
	}
}

// GetTurnoverReport returns turnover report
func GetTurnoverReport(log *slog.Logger, repo ReportsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.analytics.GetTurnoverReport"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		filter := parseAnalyticsFilter(r)

		report, err := repo.GetTurnoverStats(r.Context(), filter)
		if err != nil {
			log.Error("failed to get turnover report", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve turnover report"))
			return
		}

		log.Info("turnover report retrieved")
		render.JSON(w, r, report)
	}
}

// GetTurnoverTrend returns turnover trend
func GetTurnoverTrend(log *slog.Logger, repo ReportsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.analytics.GetTurnoverTrend"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		filter := parseAnalyticsFilter(r)

		trend, err := repo.GetTurnoverTrend(r.Context(), filter)
		if err != nil {
			log.Error("failed to get turnover trend", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve turnover trend"))
			return
		}

		log.Info("turnover trend retrieved")
		render.JSON(w, r, trend)
	}
}

// GetAttendanceReport returns attendance report
func GetAttendanceReport(log *slog.Logger, repo ReportsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.analytics.GetAttendanceReport"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		filter := parseAnalyticsFilter(r)

		report, err := repo.GetAttendanceStats(r.Context(), filter)
		if err != nil {
			log.Error("failed to get attendance report", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve attendance report"))
			return
		}

		log.Info("attendance report retrieved")
		render.JSON(w, r, report)
	}
}

// GetSalaryReport returns salary report
func GetSalaryReport(log *slog.Logger, repo ReportsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.analytics.GetSalaryReport"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		filter := parseAnalyticsFilter(r)

		report, err := repo.GetSalaryStats(r.Context(), filter)
		if err != nil {
			log.Error("failed to get salary report", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve salary report"))
			return
		}

		log.Info("salary report retrieved")
		render.JSON(w, r, report)
	}
}

// GetSalaryTrend returns salary trend
func GetSalaryTrend(log *slog.Logger, repo ReportsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.analytics.GetSalaryTrend"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		filter := parseAnalyticsFilter(r)

		trend, err := repo.GetSalaryTrend(r.Context(), filter)
		if err != nil {
			log.Error("failed to get salary trend", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve salary trend"))
			return
		}

		log.Info("salary trend retrieved")
		render.JSON(w, r, trend)
	}
}

// GetPerformanceReport returns performance report
func GetPerformanceReport(log *slog.Logger, repo ReportsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.analytics.GetPerformanceReport"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		filter := parseAnalyticsFilter(r)

		report, err := repo.GetPerformanceReport(r.Context(), filter)
		if err != nil {
			log.Error("failed to get performance report", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve performance report"))
			return
		}

		log.Info("performance report retrieved")
		render.JSON(w, r, report)
	}
}

// GetTrainingReport returns training report
func GetTrainingReport(log *slog.Logger, repo ReportsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.analytics.GetTrainingReport"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		filter := parseAnalyticsFilter(r)

		report, err := repo.GetTrainingReport(r.Context(), filter)
		if err != nil {
			log.Error("failed to get training report", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve training report"))
			return
		}

		log.Info("training report retrieved")
		render.JSON(w, r, report)
	}
}

// GetDemographicsReport returns demographics report
func GetDemographicsReport(log *slog.Logger, repo ReportsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.analytics.GetDemographicsReport"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		report, err := repo.GetDemographicsStats(r.Context())
		if err != nil {
			log.Error("failed to get demographics report", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve demographics report"))
			return
		}

		log.Info("demographics report retrieved")
		render.JSON(w, r, report)
	}
}

// GenerateCustomReport generates a custom report based on request
func GenerateCustomReport(log *slog.Logger, repo ReportsRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.analytics.GenerateCustomReport"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Parse request body
		var req hrm.CustomReportRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Warn("failed to decode request body", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request body"))
			return
		}

		// Custom reports would be built dynamically based on the metrics requested
		// For now, return a placeholder response
		response := hrm.CustomReportResponse{
			Columns: req.Metrics,
			Rows:    make([]map[string]interface{}, 0),
			Total:   0,
		}

		log.Info("custom report generated", slog.Int("metrics_count", len(req.Metrics)))
		render.JSON(w, r, response)
	}
}
