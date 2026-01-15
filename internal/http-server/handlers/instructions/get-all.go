package instructions

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/helpers"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/instruction"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type instructionGetter interface {
	GetAllInstructions(ctx context.Context, filters dto.GetAllInstructionsFilters) ([]*instruction.ResponseModel, error)
}

func GetAll(log *slog.Logger, getter instructionGetter, minioRepo helpers.MinioURLGenerator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.instructions.get-all"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		filters := dto.GetAllInstructionsFilters{}
		q := r.URL.Query()

		if typeIDStr := q.Get("type_id"); typeIDStr != "" {
			typeID, err := strconv.Atoi(typeIDStr)
			if err != nil {
				log.Warn("invalid 'type_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'type_id' parameter"))
				return
			}
			filters.TypeID = &typeID
		}

		if statusIDStr := q.Get("status_id"); statusIDStr != "" {
			statusID, err := strconv.Atoi(statusIDStr)
			if err != nil {
				log.Warn("invalid 'status_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'status_id' parameter"))
				return
			}
			filters.StatusID = &statusID
		}

		if orgIDStr := q.Get("organization_id"); orgIDStr != "" {
			orgID, err := strconv.ParseInt(orgIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'organization_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'organization_id' parameter"))
				return
			}
			filters.OrganizationID = &orgID
		}

		if contactIDStr := q.Get("responsible_contact_id"); contactIDStr != "" {
			contactID, err := strconv.ParseInt(contactIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'responsible_contact_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'responsible_contact_id' parameter"))
				return
			}
			filters.ResponsibleContactID = &contactID
		}

		if contactIDStr := q.Get("executor_contact_id"); contactIDStr != "" {
			contactID, err := strconv.ParseInt(contactIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'executor_contact_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'executor_contact_id' parameter"))
				return
			}
			filters.ExecutorContactID = &contactID
		}

		if startDateStr := q.Get("start_date"); startDateStr != "" {
			startDate, err := time.Parse("2006-01-02", startDateStr)
			if err != nil {
				log.Warn("invalid 'start_date' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'start_date' parameter (use YYYY-MM-DD format)"))
				return
			}
			filters.StartDate = &startDate
		}

		if endDateStr := q.Get("end_date"); endDateStr != "" {
			endDate, err := time.Parse("2006-01-02", endDateStr)
			if err != nil {
				log.Warn("invalid 'end_date' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'end_date' parameter (use YYYY-MM-DD format)"))
				return
			}
			filters.EndDate = &endDate
		}

		if dueDateFromStr := q.Get("due_date_from"); dueDateFromStr != "" {
			dueDateFrom, err := time.Parse("2006-01-02", dueDateFromStr)
			if err != nil {
				log.Warn("invalid 'due_date_from' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'due_date_from' parameter (use YYYY-MM-DD format)"))
				return
			}
			filters.DueDateFrom = &dueDateFrom
		}

		if dueDateToStr := q.Get("due_date_to"); dueDateToStr != "" {
			dueDateTo, err := time.Parse("2006-01-02", dueDateToStr)
			if err != nil {
				log.Warn("invalid 'due_date_to' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'due_date_to' parameter (use YYYY-MM-DD format)"))
				return
			}
			filters.DueDateTo = &dueDateTo
		}

		if nameSearch := q.Get("name"); nameSearch != "" {
			filters.NameSearch = &nameSearch
		}

		if numberSearch := q.Get("number"); numberSearch != "" {
			filters.NumberSearch = &numberSearch
		}

		documents, err := getter.GetAllInstructions(r.Context(), filters)
		if err != nil {
			log.Error("failed to get all instructions", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve instructions"))
			return
		}

		documentsWithURLs := make([]*instruction.ResponseWithURLs, 0, len(documents))
		for _, doc := range documents {
			docWithURLs := transformInstructionToResponse(r.Context(), doc, minioRepo, log)
			documentsWithURLs = append(documentsWithURLs, docWithURLs)
		}

		log.Info("successfully retrieved instructions", slog.Int("count", len(documentsWithURLs)))
		render.JSON(w, r, documentsWithURLs)
	}
}

func transformInstructionToResponse(ctx context.Context, doc *instruction.ResponseModel, minioRepo helpers.MinioURLGenerator, log *slog.Logger) *instruction.ResponseWithURLs {
	return &instruction.ResponseWithURLs{
		ID:                 doc.ID,
		Name:               doc.Name,
		Number:             doc.Number,
		DocumentDate:       doc.DocumentDate,
		Description:        doc.Description,
		Type:               doc.Type,
		Status:             doc.Status,
		ResponsibleContact: doc.ResponsibleContact,
		Organization:       doc.Organization,
		ExecutorContact:    doc.ExecutorContact,
		DueDate:            doc.DueDate,
		ParentDocument:     doc.ParentDocument,
		CreatedAt:          doc.CreatedAt,
		CreatedBy:          doc.CreatedBy,
		UpdatedAt:          doc.UpdatedAt,
		UpdatedBy:          doc.UpdatedBy,
		Files:              helpers.TransformFilesWithURLs(ctx, doc.Files, minioRepo, log),
		LinkedDocuments:    doc.LinkedDocuments,
	}
}
