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

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type addResponse struct {
	resp.Response
	ID            int64                         `json:"id"`
	UploadedFiles []fileupload.UploadedFileInfo `json:"uploaded_files,omitempty"`
}

type ShutdownAdder interface {
	AddShutdown(ctx context.Context, req dto.AddShutdownRequest, files []*multipart.FileHeader) (int64, []fileupload.UploadedFileInfo, error)
}

func Add(log *slog.Logger, adder ShutdownAdder, _ fileupload.FileUploader, _ fileupload.FileMetaSaver, _ fileupload.CategoryGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.shutdown.Add"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		var req dto.AddShutdownRequest
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

			req.StartTime, err = formparser.GetFormDateTimeRequired(r, "start_time")
			if err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid start_time"))
				return
			}

			if formparser.HasFormField(r, "end_time") {
				endTime, err := formparser.GetFormDateTime(r, "end_time")
				if err != nil {
					render.Status(r, http.StatusBadRequest)
					render.JSON(w, r, resp.BadRequest("Invalid end_time"))
					return
				}
				req.EndTime = endTime
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

			req.FileIDs, err = formparser.GetFormFileIDs(r, "file_ids")
			if err != nil {
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid file_ids"))
				return
			}

			files = r.MultipartForm.File["files"]

		} else {
			// Parse JSON (current behavior)
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

		// Additional Logic Checks
		if req.IdleDischargeVolumeThousandM3 != nil && req.EndTime == nil {
			log.Warn("validation failed: end_time is required if idle_discharge_volume is provided")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("end_time is required when providing idle_discharge_volume"))
			return
		}
		if req.EndTime != nil && !req.EndTime.After(req.StartTime) {
			log.Warn("validation failed: end_time must be after start_time")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("end_time must be after start_time"))
			return
		}

		// Call Service
		id, uploadedFiles, err := adder.AddShutdown(r.Context(), req, files)
		if err != nil {
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation (org or contact not found)")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid organization_id or reported_by_contact_id"))
				return
			}
			log.Error("failed to add shutdown", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add shutdown"))
			return
		}

		log.Info("shutdown added successfully",
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
