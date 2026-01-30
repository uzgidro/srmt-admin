package get

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
	hrmmodel "srmt-admin/internal/lib/model/hrm"
)

// PersonnelDocumentGetter defines the interface for getting personnel documents
type PersonnelDocumentGetter interface {
	GetPersonnelDocuments(ctx context.Context, filter hrm.PersonnelDocumentFilter) ([]*hrmmodel.PersonnelDocument, error)
}

// New creates a new get personnel documents handler
func New(log *slog.Logger, getter PersonnelDocumentGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.personnel_document.get.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.PersonnelDocumentFilter
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

		if docType := q.Get("document_type"); docType != "" {
			filter.DocumentType = &docType
		}

		if verifiedStr := q.Get("is_verified"); verifiedStr != "" {
			val := verifiedStr == "true"
			filter.IsVerified = &val
		}

		if expiringDaysStr := q.Get("expiring_days"); expiringDaysStr != "" {
			val, err := strconv.Atoi(expiringDaysStr)
			if err != nil {
				log.Warn("invalid 'expiring_days' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'expiring_days' parameter"))
				return
			}
			filter.ExpiringDays = &val
		}

		docs, err := getter.GetPersonnelDocuments(r.Context(), filter)
		if err != nil {
			log.Error("failed to get personnel documents", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve personnel documents"))
			return
		}

		log.Info("successfully retrieved personnel documents", slog.Int("count", len(docs)))
		render.JSON(w, r, docs)
	}
}
