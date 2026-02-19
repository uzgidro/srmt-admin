package shutdowns

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
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type editResponse struct {
	resp.Response
	UploadedFiles []fileupload.UploadedFileInfo `json:"uploaded_files,omitempty"`
}

type shutdownEditor interface {
	EditShutdown(ctx context.Context, id int64, req dto.EditShutdownRequest, files []*multipart.FileHeader) ([]fileupload.UploadedFileInfo, error)
}

func Edit(log *slog.Logger, editor shutdownEditor, _ fileupload.FileUploader, _ fileupload.FileMetaSaver, _ fileupload.CategoryGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.shutdown.Edit"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req dto.EditShutdownRequest
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
			req.OrganizationID, err = formparser.GetFormInt64(r, "organization_id")
			if err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid organization_id"))
				return
			}

			if formparser.HasFormField(r, "start_time") {
				date, err := formparser.GetFormDateTime(r, "start_time")
				if err != nil {
					render.Status(r, http.StatusBadRequest)
					render.JSON(w, r, resp.BadRequest("Invalid start_time"))
					return
				}
				req.StartTime = date
			}

			if formparser.HasFormField(r, "end_time") {
				date, err := formparser.GetFormDateTime(r, "end_time")
				if err != nil {
					render.Status(r, http.StatusBadRequest)
					render.JSON(w, r, resp.BadRequest("Invalid end_time"))
					return
				}
				req.EndTime = date
			}

			req.Reason = formparser.GetFormString(r, "reason")

			req.GenerationLossMwh, err = formparser.GetFormFloat64(r, "generation_loss")
			if err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid generation_loss"))
				return
			}

			req.ReportedByContactID, err = formparser.GetFormInt64(r, "reported_by_contact_id")
			if err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid reported_by_contact_id"))
				return
			}

			req.IdleDischargeVolumeThousandM3, err = formparser.GetFormFloat64(r, "idle_discharge_volume")
			if err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid idle_discharge_volume"))
				return
			}

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

		req.CreatedByUserID = userID

		// Update shutdown
		uploadedFiles, err := editor.EditShutdown(r.Context(), id, req, files)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("shutdown not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Shutdown not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation on update (org or contact not found)")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Organization or Contact not found"))
				return
			}
			log.Error("failed to update shutdown", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update shutdown"))
			return
		}

		log.Info("shutdown updated successfully",
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
