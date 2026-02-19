package visit

import (
	"context"
	"errors"
	"log/slog"
	"mime/multipart"
	"net/http"
	"srmt-admin/internal/lib/api/formparser"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/lib/service/fileupload"
	"srmt-admin/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type addResponse struct {
	resp.Response
	ID            int64                         `json:"id"`
	UploadedFiles []fileupload.UploadedFileInfo `json:"uploaded_files,omitempty"`
}

type visitAdder interface {
	AddVisit(ctx context.Context, req dto.AddVisitRequest, files []*multipart.FileHeader) (int64, []fileupload.UploadedFileInfo, error)
}

func Add(log *slog.Logger, adder visitAdder, _ fileupload.FileUploader, _ fileupload.FileMetaSaver, _ fileupload.CategoryGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.visit.add.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		var req dto.AddVisitRequest
		var files []*multipart.FileHeader

		// Check content type and parse accordingly
		if formparser.IsMultipartForm(r) {
			const maxUploadSize = 10 * 1024 * 1024 * 10 // 100 MB
			if err := r.ParseMultipartForm(maxUploadSize); err != nil {
				log.Error("failed to parse multipart form", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid request"))
				return
			}

			// Parse fields
			req.OrganizationID, err = formparser.GetFormInt64Required(r, "organization_id")
			if err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid organization_id"))
				return
			}

			req.VisitDate, err = formparser.GetFormDateTimeRequired(r, "visit_date")
			if err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid visit_date"))
				return
			}

			req.Description, err = formparser.GetFormStringRequired(r, "description")
			if err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("description is required"))
				return
			}

			req.ResponsibleName, err = formparser.GetFormStringRequired(r, "responsible_name")
			if err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("responsible_name is required"))
				return
			}

			req.FileIDs, err = formparser.GetFormFileIDs(r, "file_ids")
			if err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid file_ids"))
				return
			}

			files = r.MultipartForm.File["files"]

		} else {
			// Parse JSON
			if err := render.DecodeJSON(r.Body, &req); err != nil {
				log.Error("failed to decode request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid request format"))
				return
			}
		}

		req.CreatedByUserID = userID

		// Validate request
		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		// Call Service
		id, uploadedFiles, err := adder.AddVisit(r.Context(), req, files)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("organization not found", "org_id", req.OrganizationID)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Organization not found"))
				return
			}
			log.Error("failed to add visit", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add visit"))
			return
		}

		log.Info("visit added successfully",
			slog.Int64("id", id),
			slog.Int("files_linked", len(req.FileIDs)+len(uploadedFiles)),
		)

		render.Status(r, http.StatusCreated)
		response := addResponse{
			Response:      resp.Created(),
			ID:            id,
			UploadedFiles: uploadedFiles,
		}
		render.JSON(w, r, response)
	}
}
