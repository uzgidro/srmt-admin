package add

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/category"
	"srmt-admin/internal/storage"
)

type Request struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ParentID    *int64 `json:"parent_id,omitempty"`
}

type CategorySaver interface {
	AddCategory(ctx context.Context, cat category.Model) (int64, error)
	GetCategoryByID(ctx context.Context, id int64) (category.Model, error)
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

		displayName := req.Name

		if req.ParentID != nil {
			parent, err := saver.GetCategoryByID(r.Context(), *req.ParentID)
			if err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					log.Warn("parent category not found", "parent_id", *req.ParentID)
					render.Status(r, http.StatusBadRequest) // 400, т.к. клиент передал неверный ID
					render.JSON(w, r, resp.BadRequest("Parent category not found"))
					return
				}
				log.Error("failed to get parent category", sl.Err(err))
				render.Status(r, http.StatusInternalServerError)
				render.JSON(w, r, resp.InternalServerError("Failed to process category"))
				return
			}
			displayName = fmt.Sprintf("%s/%s", parent.DisplayName, req.Name)
		}

		categoryModel := category.Model{
			Name:        req.Name,
			DisplayName: displayName,
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
