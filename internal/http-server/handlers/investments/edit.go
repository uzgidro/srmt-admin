package investments

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"srmt-admin/internal/lib/api/formparser"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/service/fileupload"
	"srmt-admin/internal/storage"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type editRequest struct {
	Name     *string  `json:"name,omitempty"`
	TypeID   *int     `json:"type_id,omitempty"`
	StatusID *int     `json:"status_id,omitempty"`
	Cost     *float64 `json:"cost,omitempty"`
	Comments *string  `json:"comments,omitempty"`
	FileIDs  []int64  `json:"file_ids,omitempty"`
}

type editResponse struct {
	resp.Response
	UploadedFiles []fileupload.UploadedFileInfo `json:"uploaded_files,omitempty"`
}

type investmentEditor interface {
	EditInvestment(ctx context.Context, id int64, req dto.EditInvestmentRequest) error
	UnlinkInvestmentFiles(ctx context.Context, investmentID int64) error
	LinkInvestmentFiles(ctx context.Context, investmentID int64, fileIDs []int64) error
}

func Edit(log *slog.Logger, editor investmentEditor, uploader fileupload.FileUploader, saver fileupload.FileMetaSaver, categoryGetter fileupload.CategoryGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.investment.edit"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		id, err := formparser.GetURLParamInt64(r, "id")
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req editRequest
		var fileIDs []int64
		var shouldUpdateFiles bool
		var uploadResult *fileupload.UploadResult

		// Check content type and parse accordingly
		if formparser.IsMultipartForm(r) {
			log.Info("processing multipart/form-data request")

			// Parse request from multipart form
			req, uploadResult, err = parseMultipartEditRequest(r, log, uploader, saver, categoryGetter)
			if err != nil {
				log.Error("failed to parse multipart request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest(err.Error()))
				return
			}

			// Check if files field is present in form
			if formparser.HasFormField(r, "file_ids") || len(uploadResult.FileIDs) > 0 {
				shouldUpdateFiles = true
				// Get existing file IDs from form
				existingFileIDs, _ := formparser.GetFormFileIDs(r, "file_ids")
				// Combine uploaded + existing
				fileIDs = append(existingFileIDs, uploadResult.FileIDs...)
			}

		} else {
			log.Info("processing application/json request")

			// Parse JSON
			if err := render.DecodeJSON(r.Body, &req); err != nil {
				log.Error("failed to decode request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid request format"))
				return
			}

			// In JSON, if file_ids is present (even empty array), update files
			if req.FileIDs != nil {
				shouldUpdateFiles = true
				fileIDs = req.FileIDs
			}
		}

		// Build storage request
		storageReq := dto.EditInvestmentRequest{
			Name:     req.Name,
			TypeID:   req.TypeID,
			StatusID: req.StatusID,
			Cost:     req.Cost,
			Comments: req.Comments,
			FileIDs:  fileIDs,
		}

		// Update investment
		err = editor.EditInvestment(r.Context(), id, storageReq)
		if err != nil {
			// Cleanup uploaded files if update fails
			if uploadResult != nil {
				log.Warn("investment update failed, compensating uploaded files")
				fileupload.CompensateEntityUpload(r.Context(), log, uploader, saver, uploadResult)
			}

			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("investment not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Investment not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation on update (invalid status_id)")
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid status ID"))
				return
			}
			log.Error("failed to update investment", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update investment"))
			return
		}

		// Update file links if explicitly requested
		if shouldUpdateFiles {
			// Remove old links
			if err := editor.UnlinkInvestmentFiles(r.Context(), id); err != nil {
				log.Error("failed to unlink old files", sl.Err(err))
			}

			// Add new links (if any)
			if len(fileIDs) > 0 {
				if err := editor.LinkInvestmentFiles(r.Context(), id, fileIDs); err != nil {
					log.Error("failed to link new files", sl.Err(err))
				}
			}
		}

		log.Info("investment updated successfully",
			slog.Int64("id", id),
			slog.Bool("files_updated", shouldUpdateFiles),
			slog.Int("total_files", len(fileIDs)),
		)

		response := editResponse{
			Response: resp.OK(),
		}
		if uploadResult != nil && len(uploadResult.UploadedFiles) > 0 {
			response.UploadedFiles = uploadResult.UploadedFiles
		}
		render.JSON(w, r, response)
	}
}

// parseMultipartEditRequest parses investment data from multipart form and handles file uploads
func parseMultipartEditRequest(
	r *http.Request,
	log *slog.Logger,
	uploader fileupload.FileUploader,
	saver fileupload.FileMetaSaver,
	categoryGetter fileupload.CategoryGetter,
) (editRequest, *fileupload.UploadResult, error) {
	const op = "investments.parseMultipartEditRequest"

	// Parse optional fields
	name := formparser.GetFormString(r, "name")

	var typeID *int
	if typeIDInt64, err := formparser.GetFormInt64(r, "type_id"); err != nil {
		return editRequest{}, nil, fmt.Errorf("invalid type_id: %w", err)
	} else if typeIDInt64 != nil {
		typeIDInt := int(*typeIDInt64)
		typeID = &typeIDInt
	}

	var statusID *int
	if statusIDInt64, err := formparser.GetFormInt64(r, "status_id"); err != nil {
		return editRequest{}, nil, fmt.Errorf("invalid status_id: %w", err)
	} else if statusIDInt64 != nil {
		statusIDInt := int(*statusIDInt64)
		statusID = &statusIDInt
	}

	cost, err := formparser.GetFormFloat64(r, "cost")
	if err != nil {
		return editRequest{}, nil, fmt.Errorf("invalid cost format: %w", err)
	}

	comments := formparser.GetFormString(r, "comments")

	// Create request object
	req := editRequest{
		Name:     name,
		TypeID:   typeID,
		StatusID: statusID,
		Cost:     cost,
		Comments: comments,
	}

	// Process file uploads
	uploadResult, err := fileupload.ProcessFormFiles(
		r.Context(),
		r,
		log,
		uploader,
		saver,
		categoryGetter,
		"investments", // category name for MinIO path
		"Инвестиции",  // category display name
		time.Now(),    // For edits, use current time
	)
	if err != nil {
		return editRequest{}, nil, fmt.Errorf("%s: failed to process file uploads: %w", op, err)
	}

	log.Info("multipart edit form parsed successfully",
		slog.Int("uploaded_files", len(uploadResult.FileIDs)),
	)

	return req, uploadResult, nil
}
