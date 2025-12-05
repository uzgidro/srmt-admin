package add

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/file"
	"srmt-admin/internal/lib/service/fileupload"
	"srmt-admin/internal/storage"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// Request - JSON DTO хендлера
type Request struct {
	Name            string     `json:"name" validate:"required"`
	Email           *string    `json:"email,omitempty" validate:"omitempty,email"`
	Phone           *string    `json:"phone,omitempty"`
	IPPhone         *string    `json:"ip_phone,omitempty"`
	DOB             *time.Time `json:"dob,omitempty"`
	ExternalOrgName *string    `json:"external_organization_name,omitempty"`
	OrganizationID  *int64     `json:"organization_id,omitempty"`
	DepartmentID    *int64     `json:"department_id,omitempty"`
	PositionID      *int64     `json:"position_id,omitempty"`
}

type Response struct {
	resp.Response
	ID int64 `json:"id"`
}

type FileUploader interface {
	UploadFile(ctx context.Context, objectName string, reader io.Reader, size int64, contentType string) error
	DeleteFile(ctx context.Context, objectName string) error
}

type FileMetaSaver interface {
	AddFile(ctx context.Context, fileData file.Model) (int64, error)
	GetCategoryByName(ctx context.Context, categoryName string) (fileupload.CategoryModel, error)
}

// ContactAdder - интерфейс репозитория
type ContactAdder interface {
	AddContact(ctx context.Context, req dto.AddContactRequest) (int64, error)
}

func New(log *slog.Logger, adder ContactAdder, uploader FileUploader, fileSaver FileMetaSaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.contact.add.New"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		contentType := r.Header.Get("Content-Type")
		isMultipart := strings.Contains(contentType, "multipart/form-data")

		var req Request
		var iconID *int64

		if isMultipart {
			// Parse multipart form
			const maxUploadSize = 10 * 1024 * 1024 // 10 MB for icon
			r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
			if err := r.ParseMultipartForm(maxUploadSize); err != nil {
				log.Error("failed to parse multipart form", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid request or file is too large"))
				return
			}

			// Parse form fields
			req.Name = r.FormValue("name")
			if email := r.FormValue("email"); email != "" {
				req.Email = &email
			}
			if phone := r.FormValue("phone"); phone != "" {
				req.Phone = &phone
			}
			if ipPhone := r.FormValue("ip_phone"); ipPhone != "" {
				req.IPPhone = &ipPhone
			}
			if dobStr := r.FormValue("dob"); dobStr != "" {
				dob, err := time.Parse(time.DateOnly, dobStr)
				if err != nil {
					log.Error("invalid dob format", sl.Err(err))
					render.Status(r, http.StatusBadRequest)
					render.JSON(w, r, resp.BadRequest("Invalid dob format, use RFC3339"))
					return
				}
				req.DOB = &dob
			}
			if extOrg := r.FormValue("external_organization_name"); extOrg != "" {
				req.ExternalOrgName = &extOrg
			}
			if orgIDStr := r.FormValue("organization_id"); orgIDStr != "" {
				orgID, err := strconv.ParseInt(orgIDStr, 10, 64)
				if err != nil {
					log.Error("invalid organization_id", sl.Err(err))
					render.Status(r, http.StatusBadRequest)
					render.JSON(w, r, resp.BadRequest("Invalid organization_id"))
					return
				}
				req.OrganizationID = &orgID
			}
			if deptIDStr := r.FormValue("department_id"); deptIDStr != "" {
				deptID, err := strconv.ParseInt(deptIDStr, 10, 64)
				if err != nil {
					log.Error("invalid department_id", sl.Err(err))
					render.Status(r, http.StatusBadRequest)
					render.JSON(w, r, resp.BadRequest("Invalid department_id"))
					return
				}
				req.DepartmentID = &deptID
			}
			if posIDStr := r.FormValue("position_id"); posIDStr != "" {
				posID, err := strconv.ParseInt(posIDStr, 10, 64)
				if err != nil {
					log.Error("invalid position_id", sl.Err(err))
					render.Status(r, http.StatusBadRequest)
					render.JSON(w, r, resp.BadRequest("Invalid position_id"))
					return
				}
				req.PositionID = &posID
			}

			// Handle icon file upload if present
			formFile, handler, err := r.FormFile("icon")
			if err == nil {
				defer formFile.Close()

				// Get or create "icon" category
				cat, err := fileSaver.GetCategoryByName(r.Context(), "icon")
				if err != nil {
					log.Error("failed to get icon category", sl.Err(err))
					render.Status(r, http.StatusInternalServerError)
					render.JSON(w, r, resp.InternalServerError("Failed to process icon"))
					return
				}

				// Generate unique object key
				objectKey := fmt.Sprintf("icon/%s%s",
					uuid.New().String(),
					filepath.Ext(handler.Filename),
				)

				// Upload to storage
				err = uploader.UploadFile(r.Context(), objectKey, formFile, handler.Size, handler.Header.Get("Content-Type"))
				if err != nil {
					log.Error("failed to upload icon to storage", sl.Err(err))
					render.Status(r, http.StatusInternalServerError)
					render.JSON(w, r, resp.InternalServerError("Failed to upload icon"))
					return
				}

				// Save file metadata
				fileModel := file.Model{
					FileName:   handler.Filename,
					ObjectKey:  objectKey,
					CategoryID: cat.GetID(),
					MimeType:   handler.Header.Get("Content-Type"),
					SizeBytes:  handler.Size,
					CreatedAt:  time.Now(),
				}

				fileID, err := fileSaver.AddFile(r.Context(), fileModel)
				if err != nil {
					log.Error("failed to save icon metadata", sl.Err(err))
					// Compensate: delete uploaded file
					if delErr := uploader.DeleteFile(r.Context(), objectKey); delErr != nil {
						log.Error("compensation failed: could not delete orphaned icon", sl.Err(delErr))
					}
					render.Status(r, http.StatusInternalServerError)
					render.JSON(w, r, resp.InternalServerError("Failed to save icon"))
					return
				}

				iconID = &fileID
			} else if err != http.ErrMissingFile {
				log.Error("failed to get icon file", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid icon file"))
				return
			}

		} else {
			// Parse JSON
			if err := render.DecodeJSON(r.Body, &req); err != nil {
				log.Error("failed to decode request", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid request format"))
				return
			}
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		// Маппинг в DTO хранилища
		storageReq := dto.AddContactRequest{
			Name:            req.Name,
			Email:           req.Email,
			Phone:           req.Phone,
			IPPhone:         req.IPPhone,
			DOB:             req.DOB,
			ExternalOrgName: req.ExternalOrgName,
			IconID:          iconID,
			OrganizationID:  req.OrganizationID,
			DepartmentID:    req.DepartmentID,
			PositionID:      req.PositionID,
		}

		id, err := adder.AddContact(r.Context(), storageReq)
		if err != nil {
			if errors.Is(err, storage.ErrDuplicate) {
				log.Warn("duplicate contact data", "name", req.Name)
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.BadRequest("Contact with this email or phone already exists"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("FK violation", "org_id", req.OrganizationID)
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid organization, department or position"))
				return
			}
			log.Error("failed to add contact", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add contact"))
			return
		}

		log.Info("contact added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, Response{Response: resp.OK(), ID: id})
	}
}
