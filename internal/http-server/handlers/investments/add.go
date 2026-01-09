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
	"srmt-admin/internal/lib/service/auth"
	"srmt-admin/internal/lib/service/fileupload"
	"srmt-admin/internal/storage"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
)

type addRequest struct {
	Name     string  `json:"name" validate:"required"`
	TypeID   int     `json:"type_id" validate:"required,min=1"`
	StatusID int     `json:"status_id" validate:"required,min=1"`
	Cost     float64 `json:"cost" validate:"gte=0"`
	Comments *string `json:"comments,omitempty"`
	FileIDs  []int64 `json:"file_ids,omitempty"`
}

type addResponse struct {
	resp.Response
	ID            int64                         `json:"id"`
	UploadedFiles []fileupload.UploadedFileInfo `json:"uploaded_files,omitempty"`
}

type investmentAdder interface {
	AddInvestment(ctx context.Context, req dto.AddInvestmentRequest, createdByID int64) (int64, error)
	LinkInvestmentFiles(ctx context.Context, investmentID int64, fileIDs []int64) error
}

func Add(log *slog.Logger, adder investmentAdder, uploader fileupload.FileUploader, saver fileupload.FileMetaSaver, categoryGetter fileupload.CategoryGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.investment.add"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		userID, err := auth.GetUserID(r.Context())
		if err != nil {
			log.Error("failed to get user id from context", sl.Err(err))
			render.Status(r, http.StatusUnauthorized)
			render.JSON(w, r, resp.Unauthorized("Not authenticated"))
			return
		}

		var req addRequest
		var fileIDs []int64
		var uploadResult *fileupload.UploadResult

		// Check content type and parse accordingly
		if formparser.IsMultipartForm(r) {
			log.Info("processing multipart/form-data request")

			// Parse request from multipart form
			req, uploadResult, err = parseMultipartAddRequest(r, log, uploader, saver, categoryGetter)
			if err != nil {
				log.Error("failed to parse multipart request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest(err.Error()))
				return
			}

			// Combine uploaded files + existing file IDs
			existingFileIDs, _ := formparser.GetFormFileIDs(r, "file_ids")
			fileIDs = append(existingFileIDs, uploadResult.FileIDs...)

		} else {
			log.Info("processing application/json request")

			// Parse JSON
			if err := render.DecodeJSON(r.Body, &req); err != nil {
				log.Error("failed to decode request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid request format"))
				return
			}

			fileIDs = req.FileIDs
		}

		// Validate request
		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))

			// Cleanup uploaded files if validation fails
			if uploadResult != nil {
				fileupload.CompensateEntityUpload(r.Context(), log, uploader, saver, uploadResult)
			}

			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		// Create DTO for storage
		storageReq := dto.AddInvestmentRequest{
			Name:     req.Name,
			TypeID:   req.TypeID,
			StatusID: req.StatusID,
			Cost:     req.Cost,
			Comments: req.Comments,
			FileIDs:  fileIDs,
		}

		// Create investment
		id, err := adder.AddInvestment(r.Context(), storageReq, userID)
		if err != nil {
			// Cleanup uploaded files if investment creation fails
			if uploadResult != nil {
				log.Warn("investment creation failed, compensating uploaded files")
				fileupload.CompensateEntityUpload(r.Context(), log, uploader, saver, uploadResult)
			}

			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("foreign key violation", slog.Int("status_id", req.StatusID))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid status ID"))
				return
			}
			log.Error("failed to add investment", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add investment"))
			return
		}

		// Link files if provided
		if len(fileIDs) > 0 {
			if err := adder.LinkInvestmentFiles(r.Context(), id, fileIDs); err != nil {
				log.Error("failed to link files", sl.Err(err))
				// Don't fail the request, just log the error
			}
		}

		uploadedFilesCount := 0
		if uploadResult != nil {
			uploadedFilesCount = len(uploadResult.FileIDs)
		}
		log.Info("investment added successfully",
			slog.Int64("id", id),
			slog.Int("total_files", len(fileIDs)),
			slog.Int("uploaded_files", uploadedFilesCount),
		)

		render.Status(r, http.StatusCreated)
		response := addResponse{
			Response: resp.Created(),
			ID:       id,
		}
		if uploadResult != nil && len(uploadResult.UploadedFiles) > 0 {
			response.UploadedFiles = uploadResult.UploadedFiles
		}
		render.JSON(w, r, response)
	}
}

// parseMultipartAddRequest parses investment data from multipart form and handles file uploads
func parseMultipartAddRequest(
	r *http.Request,
	log *slog.Logger,
	uploader fileupload.FileUploader,
	saver fileupload.FileMetaSaver,
	categoryGetter fileupload.CategoryGetter,
) (addRequest, *fileupload.UploadResult, error) {
	const op = "investments.parseMultipartAddRequest"

	// Parse name (required)
	name, err := formparser.GetFormStringRequired(r, "name")
	if err != nil {
		return addRequest{}, nil, err
	}

	// Parse type_id (required)
	typeIDInt64, err := formparser.GetFormInt64Required(r, "type_id")
	if err != nil {
		return addRequest{}, nil, fmt.Errorf("invalid or missing type_id: %w", err)
	}
	typeID := int(typeIDInt64)

	// Parse status_id (required)
	statusIDInt64, err := formparser.GetFormInt64Required(r, "status_id")
	if err != nil {
		return addRequest{}, nil, fmt.Errorf("invalid or missing status_id: %w", err)
	}
	statusID := int(statusIDInt64)

	// Parse cost (required, default to 0.0)
	cost, err := formparser.GetFormFloat64(r, "cost")
	if err != nil {
		return addRequest{}, nil, fmt.Errorf("invalid cost format: %w", err)
	}
	if cost == nil {
		defaultCost := 0.0
		cost = &defaultCost
	}

	// Parse comments (optional)
	comments := formparser.GetFormString(r, "comments")

	// Create request object
	req := addRequest{
		Name:     name,
		TypeID:   typeID,
		StatusID: statusID,
		Cost:     *cost,
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
		"Инвестиции",  // category display name (Russian)
		time.Now(),    // upload date
	)
	if err != nil {
		return addRequest{}, nil, fmt.Errorf("%s: failed to process file uploads: %w", op, err)
	}

	log.Info("multipart form parsed successfully",
		slog.Int("uploaded_files", len(uploadResult.FileIDs)),
	)

	return req, uploadResult, nil
}
