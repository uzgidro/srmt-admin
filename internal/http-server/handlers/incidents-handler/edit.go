package incidents_handler

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
	"srmt-admin/internal/lib/service/fileupload"
	"srmt-admin/internal/storage"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type editResponse struct {
	resp.Response
	UploadedFiles []fileupload.UploadedFileInfo `json:"uploaded_files,omitempty"`
}

type incidentEditor interface {
	EditIncident(ctx context.Context, id int64, req dto.EditIncidentRequest, files []*multipart.FileHeader) ([]fileupload.UploadedFileInfo, error)
}

func Edit(log *slog.Logger, editor incidentEditor, _ fileupload.FileUploader, _ fileupload.FileMetaSaver, _ fileupload.CategoryGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.incident.update.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req dto.EditIncidentRequest
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
			// organization_id (optional)
			req.OrganizationID, err = formparser.GetFormInt64(r, "organization_id")
			if err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid organization_id"))
				return
			}

			if formparser.HasFormField(r, "incident_time") {
				date, err := formparser.GetFormDateTime(r, "incident_time")
				if err != nil {
					render.Status(r, http.StatusBadRequest)
					render.JSON(w, r, resp.BadRequest("Invalid incident_time"))
					return
				}
				req.IncidentTime = date
			}

			// description (optional)
			req.Description = formparser.GetFormString(r, "description")

			// file_ids (optional)
			// If field is present, we parse it. If distinct from missing, we set req.FileIDs.
			if formparser.HasFormField(r, "file_ids") {
				fIDs, err := formparser.GetFormFileIDs(r, "file_ids")
				if err != nil {
					render.Status(r, http.StatusBadRequest)
					render.JSON(w, r, resp.BadRequest("Invalid file_ids"))
					return
				}
				req.FileIDs = fIDs
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

		// Update incident
		uploadedFiles, err := editor.EditIncident(r.Context(), id, req, files)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("incident not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Incident not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation on update (org_id not found)")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Organization not found"))
				return
			}
			log.Error("failed to update incident", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update incident"))
			return
		}

		log.Info("incident updated successfully",
			slog.Int64("id", id),
			slog.Bool("files_updated", req.FileIDs != nil || len(files) > 0),
		)

		response := editResponse{
			Response:      resp.OK(),
			UploadedFiles: uploadedFiles,
		}
		render.JSON(w, r, response)
	}
}
