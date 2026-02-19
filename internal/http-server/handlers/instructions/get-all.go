package instructions

import (
	"context"
	"log/slog"
	"net/http"

	"srmt-admin/internal/lib/api/formparser"
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

		if typeIDVal, err := formparser.GetFormInt64(r, "type_id"); err != nil {
			log.Warn("invalid 'type_id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'type_id' parameter"))
			return
		} else if typeIDVal != nil {
			val := int(*typeIDVal)
			filters.TypeID = &val
		}

		if statusIDVal, err := formparser.GetFormInt64(r, "status_id"); err != nil {
			log.Warn("invalid 'status_id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'status_id' parameter"))
			return
		} else if statusIDVal != nil {
			val := int(*statusIDVal)
			filters.StatusID = &val
		}

		if orgIDVal, err := formparser.GetFormInt64(r, "organization_id"); err != nil {
			log.Warn("invalid 'organization_id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'organization_id' parameter"))
			return
		} else {
			filters.OrganizationID = orgIDVal
		}

		if contactIDVal, err := formparser.GetFormInt64(r, "responsible_contact_id"); err != nil {
			log.Warn("invalid 'responsible_contact_id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'responsible_contact_id' parameter"))
			return
		} else {
			filters.ResponsibleContactID = contactIDVal
		}

		if contactIDVal, err := formparser.GetFormInt64(r, "executor_contact_id"); err != nil {
			log.Warn("invalid 'executor_contact_id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'executor_contact_id' parameter"))
			return
		} else {
			filters.ExecutorContactID = contactIDVal
		}

		if startDateVal, err := formparser.GetFormDate(r, "start_date"); err != nil {
			log.Warn("invalid 'start_date' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'start_date' parameter (use YYYY-MM-DD format)"))
			return
		} else {
			filters.StartDate = startDateVal
		}

		if endDateVal, err := formparser.GetFormDate(r, "end_date"); err != nil {
			log.Warn("invalid 'end_date' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'end_date' parameter (use YYYY-MM-DD format)"))
			return
		} else {
			filters.EndDate = endDateVal
		}

		if dueDateFromVal, err := formparser.GetFormDate(r, "due_date_from"); err != nil {
			log.Warn("invalid 'due_date_from' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'due_date_from' parameter (use YYYY-MM-DD format)"))
			return
		} else {
			filters.DueDateFrom = dueDateFromVal
		}

		if dueDateToVal, err := formparser.GetFormDate(r, "due_date_to"); err != nil {
			log.Warn("invalid 'due_date_to' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'due_date_to' parameter (use YYYY-MM-DD format)"))
			return
		} else {
			filters.DueDateTo = dueDateToVal
		}

		filters.NameSearch = formparser.GetFormString(r, "name")
		filters.NumberSearch = formparser.GetFormString(r, "number")

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
