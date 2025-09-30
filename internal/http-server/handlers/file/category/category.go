package category

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/category"
	"srmt-admin/internal/storage"
	"strings"
)

type Request struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description,omitempty"`
	ParentID    *int64 `json:"parent_id,omitempty"`
}

type CategorySaver interface {
	AddCategory(ctx context.Context, cat category.Model) (int64, error)
}

func New(log *slog.Logger, saver CategorySaver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.file.category.New"
		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request body", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if strings.TrimSpace(req.Name) == "" || strings.TrimSpace(req.DisplayName) == "" {
			log.Error("required fields are empty", slog.String("name", req.Name), slog.String("display_name", req.DisplayName))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Fields 'name' and 'display_name' are required"))
			return
		}

		categoryModel := category.Model{
			Name:        req.Name,
			DisplayName: req.DisplayName,
			Description: req.Description,
			ParentID:    req.ParentID,
		}

		id, err := saver.AddCategory(r.Context(), categoryModel)
		if err != nil {
			if errors.Is(err, storage.ErrDuplicate) {
				log.Warn("category with this name already exists", sl.Err(err))
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Category with this name already exists"))
				return
			}
			log.Error("failed to add category", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add category"))
			return
		}

		log.Info("category added successfully", slog.Int64("id", id))

		render.Status(r, http.StatusCreated)
		render.JSON(w, r, map[string]interface{}{
			"message": "Category created successfully",
			"id":      id,
		})
	}
}
