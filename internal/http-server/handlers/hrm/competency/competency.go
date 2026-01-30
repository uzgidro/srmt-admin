package competency

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto/hrm"
	"srmt-admin/internal/lib/logger/sl"
	hrmmodel "srmt-admin/internal/lib/model/hrm"
	"srmt-admin/internal/storage"
)

// --- Repository Interfaces ---

type CategoryRepository interface {
	AddCompetencyCategory(ctx context.Context, req hrm.AddCompetencyCategoryRequest) (int, error)
	GetCompetencyCategoryByID(ctx context.Context, id int) (*hrmmodel.CompetencyCategory, error)
	GetCompetencyCategories(ctx context.Context) ([]*hrmmodel.CompetencyCategory, error)
	EditCompetencyCategory(ctx context.Context, id int, req hrm.EditCompetencyCategoryRequest) error
	DeleteCompetencyCategory(ctx context.Context, id int) error
}

type CompetencyRepository interface {
	AddCompetency(ctx context.Context, req hrm.AddCompetencyRequest) (int, error)
	GetCompetencyByID(ctx context.Context, id int) (*hrmmodel.Competency, error)
	GetCompetencies(ctx context.Context, filter hrm.CompetencyFilter) ([]*hrmmodel.Competency, error)
	EditCompetency(ctx context.Context, id int, req hrm.EditCompetencyRequest) error
	DeleteCompetency(ctx context.Context, id int) error
}

type LevelRepository interface {
	AddCompetencyLevel(ctx context.Context, req hrm.AddCompetencyLevelRequest) (int, error)
	GetCompetencyLevels(ctx context.Context, competencyID int) ([]*hrmmodel.CompetencyLevel, error)
	EditCompetencyLevel(ctx context.Context, id int, req hrm.EditCompetencyLevelRequest) error
	DeleteCompetencyLevel(ctx context.Context, id int) error
}

type MatrixRepository interface {
	AddCompetencyMatrix(ctx context.Context, req hrm.AddCompetencyMatrixRequest) (int64, error)
	GetCompetencyMatrix(ctx context.Context, filter hrm.CompetencyMatrixFilter) ([]*hrmmodel.CompetencyMatrix, error)
	EditCompetencyMatrix(ctx context.Context, id int64, req hrm.EditCompetencyMatrixRequest) error
	DeleteCompetencyMatrix(ctx context.Context, id int64) error
}

type AssessmentRepository interface {
	AddCompetencyAssessment(ctx context.Context, req hrm.AddAssessmentRequest) (int64, error)
	GetCompetencyAssessmentByID(ctx context.Context, id int64) (*hrmmodel.CompetencyAssessment, error)
	GetCompetencyAssessments(ctx context.Context, filter hrm.AssessmentFilter) ([]*hrmmodel.CompetencyAssessment, error)
	StartCompetencyAssessment(ctx context.Context, id int64) error
	CompleteCompetencyAssessment(ctx context.Context, id int64, req hrm.CompleteAssessmentRequest) error
	DeleteCompetencyAssessment(ctx context.Context, id int64) error
}

type ScoreRepository interface {
	AddCompetencyScore(ctx context.Context, req hrm.AddScoreRequest) (int64, error)
	BulkAddCompetencyScores(ctx context.Context, req hrm.BulkScoresRequest) error
	GetCompetencyScores(ctx context.Context, filter hrm.ScoreFilter) ([]*hrmmodel.CompetencyScore, error)
	EditCompetencyScore(ctx context.Context, id int64, req hrm.EditScoreRequest) error
	DeleteCompetencyScore(ctx context.Context, id int64) error
}

// IDResponse represents a response with ID
type IDResponse struct {
	resp.Response
	ID int64 `json:"id"`
}

// --- Category Handlers ---

func GetCategories(log *slog.Logger, repo CategoryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.GetCategories"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		categories, err := repo.GetCompetencyCategories(r.Context())
		if err != nil {
			log.Error("failed to get categories", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve categories"))
			return
		}

		log.Info("successfully retrieved categories", slog.Int("count", len(categories)))
		render.JSON(w, r, categories)
	}
}

func GetCategoryByID(log *slog.Logger, repo CategoryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.GetCategoryByID"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		category, err := repo.GetCompetencyCategoryByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("category not found", slog.Int("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Category not found"))
				return
			}
			log.Error("failed to get category", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve category"))
			return
		}

		render.JSON(w, r, category)
	}
}

func AddCategory(log *slog.Logger, repo CategoryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.AddCategory"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddCompetencyCategoryRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		id, err := repo.AddCompetencyCategory(r.Context(), req)
		if err != nil {
			log.Error("failed to add category", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add category"))
			return
		}

		log.Info("category added", slog.Int("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: int64(id)})
	}
}

func EditCategory(log *slog.Logger, repo CategoryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.EditCategory"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.EditCompetencyCategoryRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.EditCompetencyCategory(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("category not found", slog.Int("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Category not found"))
				return
			}
			log.Error("failed to update category", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update category"))
			return
		}

		log.Info("category updated", slog.Int("id", id))
		render.JSON(w, r, resp.OK())
	}
}

func DeleteCategory(log *slog.Logger, repo CategoryRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.DeleteCategory"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteCompetencyCategory(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("category not found", slog.Int("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Category not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("category has dependencies", slog.Int("id", id))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Cannot delete: category is in use"))
				return
			}
			log.Error("failed to delete category", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete category"))
			return
		}

		log.Info("category deleted", slog.Int("id", id))
		render.Status(r, http.StatusNoContent)
	}
}

// --- Competency Handlers ---

func GetCompetencies(log *slog.Logger, repo CompetencyRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.GetCompetencies"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.CompetencyFilter
		q := r.URL.Query()

		if catIDStr := q.Get("category_id"); catIDStr != "" {
			val, err := strconv.Atoi(catIDStr)
			if err != nil {
				log.Warn("invalid 'category_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'category_id' parameter"))
				return
			}
			filter.CategoryID = &val
		}

		if isActiveStr := q.Get("is_active"); isActiveStr != "" {
			val := isActiveStr == "true"
			filter.IsActive = &val
		}

		if search := q.Get("search"); search != "" {
			filter.Search = &search
		}

		competencies, err := repo.GetCompetencies(r.Context(), filter)
		if err != nil {
			log.Error("failed to get competencies", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve competencies"))
			return
		}

		log.Info("successfully retrieved competencies", slog.Int("count", len(competencies)))
		render.JSON(w, r, competencies)
	}
}

func GetCompetencyByID(log *slog.Logger, repo CompetencyRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.GetCompetencyByID"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		competency, err := repo.GetCompetencyByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("competency not found", slog.Int("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Competency not found"))
				return
			}
			log.Error("failed to get competency", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve competency"))
			return
		}

		render.JSON(w, r, competency)
	}
}

func AddCompetency(log *slog.Logger, repo CompetencyRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.AddCompetency"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddCompetencyRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		id, err := repo.AddCompetency(r.Context(), req)
		if err != nil {
			log.Error("failed to add competency", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add competency"))
			return
		}

		log.Info("competency added", slog.Int("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: int64(id)})
	}
}

func EditCompetency(log *slog.Logger, repo CompetencyRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.EditCompetency"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.EditCompetencyRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.EditCompetency(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("competency not found", slog.Int("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Competency not found"))
				return
			}
			log.Error("failed to update competency", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update competency"))
			return
		}

		log.Info("competency updated", slog.Int("id", id))
		render.JSON(w, r, resp.OK())
	}
}

func DeleteCompetency(log *slog.Logger, repo CompetencyRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.DeleteCompetency"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteCompetency(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("competency not found", slog.Int("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Competency not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("competency has dependencies", slog.Int("id", id))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Cannot delete: competency is in use"))
				return
			}
			log.Error("failed to delete competency", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete competency"))
			return
		}

		log.Info("competency deleted", slog.Int("id", id))
		render.Status(r, http.StatusNoContent)
	}
}

// --- Level Handlers ---

func GetLevels(log *slog.Logger, repo LevelRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.GetLevels"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		compIDStr := chi.URLParam(r, "competencyId")
		compID, err := strconv.Atoi(compIDStr)
		if err != nil {
			log.Warn("invalid 'competencyId' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'competencyId' parameter"))
			return
		}

		levels, err := repo.GetCompetencyLevels(r.Context(), compID)
		if err != nil {
			log.Error("failed to get levels", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve levels"))
			return
		}

		log.Info("successfully retrieved levels", slog.Int("count", len(levels)))
		render.JSON(w, r, levels)
	}
}

func AddLevel(log *slog.Logger, repo LevelRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.AddLevel"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddCompetencyLevelRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		id, err := repo.AddCompetencyLevel(r.Context(), req)
		if err != nil {
			if errors.Is(err, storage.ErrUniqueViolation) {
				log.Warn("duplicate level")
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Level already exists for this competency"))
				return
			}
			log.Error("failed to add level", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add level"))
			return
		}

		log.Info("level added", slog.Int("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: int64(id)})
	}
}

func EditLevel(log *slog.Logger, repo LevelRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.EditLevel"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.EditCompetencyLevelRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.EditCompetencyLevel(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("level not found", slog.Int("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Level not found"))
				return
			}
			log.Error("failed to update level", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update level"))
			return
		}

		log.Info("level updated", slog.Int("id", id))
		render.JSON(w, r, resp.OK())
	}
}

func DeleteLevel(log *slog.Logger, repo LevelRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.DeleteLevel"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteCompetencyLevel(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("level not found", slog.Int("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Level not found"))
				return
			}
			log.Error("failed to delete level", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete level"))
			return
		}

		log.Info("level deleted", slog.Int("id", id))
		render.Status(r, http.StatusNoContent)
	}
}

// --- Matrix Handlers ---

func GetMatrix(log *slog.Logger, repo MatrixRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.GetMatrix"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.CompetencyMatrixFilter
		q := r.URL.Query()

		if posIDStr := q.Get("position_id"); posIDStr != "" {
			val, err := strconv.ParseInt(posIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'position_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'position_id' parameter"))
				return
			}
			filter.PositionID = &val
		}

		if compIDStr := q.Get("competency_id"); compIDStr != "" {
			val, err := strconv.Atoi(compIDStr)
			if err != nil {
				log.Warn("invalid 'competency_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'competency_id' parameter"))
				return
			}
			filter.CompetencyID = &val
		}

		if isMandatoryStr := q.Get("is_mandatory"); isMandatoryStr != "" {
			val := isMandatoryStr == "true"
			filter.IsMandatory = &val
		}

		matrix, err := repo.GetCompetencyMatrix(r.Context(), filter)
		if err != nil {
			log.Error("failed to get matrix", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve matrix"))
			return
		}

		log.Info("successfully retrieved matrix", slog.Int("count", len(matrix)))
		render.JSON(w, r, matrix)
	}
}

func AddMatrix(log *slog.Logger, repo MatrixRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.AddMatrix"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddCompetencyMatrixRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		id, err := repo.AddCompetencyMatrix(r.Context(), req)
		if err != nil {
			if errors.Is(err, storage.ErrUniqueViolation) {
				log.Warn("duplicate matrix entry")
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Matrix entry already exists"))
				return
			}
			log.Error("failed to add matrix entry", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add matrix entry"))
			return
		}

		log.Info("matrix entry added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

func EditMatrix(log *slog.Logger, repo MatrixRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.EditMatrix"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.EditCompetencyMatrixRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.EditCompetencyMatrix(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("matrix entry not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Matrix entry not found"))
				return
			}
			log.Error("failed to update matrix entry", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update matrix entry"))
			return
		}

		log.Info("matrix entry updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

func DeleteMatrix(log *slog.Logger, repo MatrixRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.DeleteMatrix"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteCompetencyMatrix(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("matrix entry not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Matrix entry not found"))
				return
			}
			log.Error("failed to delete matrix entry", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete matrix entry"))
			return
		}

		log.Info("matrix entry deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}

// --- Assessment Handlers ---

func GetAssessments(log *slog.Logger, repo AssessmentRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.GetAssessments"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.AssessmentFilter
		q := r.URL.Query()

		if empIDStr := q.Get("employee_id"); empIDStr != "" {
			val, err := strconv.ParseInt(empIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'employee_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'employee_id' parameter"))
				return
			}
			filter.EmployeeID = &val
		}

		if assessorIDStr := q.Get("assessor_id"); assessorIDStr != "" {
			val, err := strconv.ParseInt(assessorIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'assessor_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'assessor_id' parameter"))
				return
			}
			filter.AssessorID = &val
		}

		if aType := q.Get("assessment_type"); aType != "" {
			filter.AssessmentType = &aType
		}

		if status := q.Get("status"); status != "" {
			filter.Status = &status
		}

		if limitStr := q.Get("limit"); limitStr != "" {
			val, _ := strconv.Atoi(limitStr)
			filter.Limit = val
		}

		if offsetStr := q.Get("offset"); offsetStr != "" {
			val, _ := strconv.Atoi(offsetStr)
			filter.Offset = val
		}

		assessments, err := repo.GetCompetencyAssessments(r.Context(), filter)
		if err != nil {
			log.Error("failed to get assessments", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve assessments"))
			return
		}

		log.Info("successfully retrieved assessments", slog.Int("count", len(assessments)))
		render.JSON(w, r, assessments)
	}
}

func GetAssessmentByID(log *slog.Logger, repo AssessmentRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.GetAssessmentByID"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		assessment, err := repo.GetCompetencyAssessmentByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("assessment not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Assessment not found"))
				return
			}
			log.Error("failed to get assessment", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve assessment"))
			return
		}

		render.JSON(w, r, assessment)
	}
}

func AddAssessment(log *slog.Logger, repo AssessmentRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.AddAssessment"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddAssessmentRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		id, err := repo.AddCompetencyAssessment(r.Context(), req)
		if err != nil {
			log.Error("failed to add assessment", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add assessment"))
			return
		}

		log.Info("assessment added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

func StartAssessment(log *slog.Logger, repo AssessmentRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.StartAssessment"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.StartCompetencyAssessment(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("assessment not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Assessment not found"))
				return
			}
			log.Error("failed to start assessment", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to start assessment"))
			return
		}

		log.Info("assessment started", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

func CompleteAssessment(log *slog.Logger, repo AssessmentRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.CompleteAssessment"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.CompleteAssessmentRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.CompleteCompetencyAssessment(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("assessment not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Assessment not found"))
				return
			}
			log.Error("failed to complete assessment", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to complete assessment"))
			return
		}

		log.Info("assessment completed", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

func DeleteAssessment(log *slog.Logger, repo AssessmentRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.DeleteAssessment"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteCompetencyAssessment(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("assessment not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Assessment not found"))
				return
			}
			if errors.Is(err, storage.ErrForeignKeyViolation) {
				log.Warn("assessment has dependencies", slog.Int64("id", id))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Cannot delete: assessment has scores"))
				return
			}
			log.Error("failed to delete assessment", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete assessment"))
			return
		}

		log.Info("assessment deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}

// --- Score Handlers ---

func GetScores(log *slog.Logger, repo ScoreRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.GetScores"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var filter hrm.ScoreFilter
		q := r.URL.Query()

		if assessmentIDStr := q.Get("assessment_id"); assessmentIDStr != "" {
			val, err := strconv.ParseInt(assessmentIDStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'assessment_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'assessment_id' parameter"))
				return
			}
			filter.AssessmentID = &val
		}

		if compIDStr := q.Get("competency_id"); compIDStr != "" {
			val, err := strconv.Atoi(compIDStr)
			if err != nil {
				log.Warn("invalid 'competency_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'competency_id' parameter"))
				return
			}
			filter.CompetencyID = &val
		}

		scores, err := repo.GetCompetencyScores(r.Context(), filter)
		if err != nil {
			log.Error("failed to get scores", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve scores"))
			return
		}

		log.Info("successfully retrieved scores", slog.Int("count", len(scores)))
		render.JSON(w, r, scores)
	}
}

func AddScore(log *slog.Logger, repo ScoreRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.AddScore"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.AddScoreRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		id, err := repo.AddCompetencyScore(r.Context(), req)
		if err != nil {
			if errors.Is(err, storage.ErrUniqueViolation) {
				log.Warn("duplicate score")
				render.Status(r, http.StatusConflict)
				render.JSON(w, r, resp.Conflict("Score already exists for this assessment and competency"))
				return
			}
			log.Error("failed to add score", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add score"))
			return
		}

		log.Info("score added", slog.Int64("id", id))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, IDResponse{Response: resp.OK(), ID: id})
	}
}

func BulkAddScores(log *slog.Logger, repo ScoreRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.BulkAddScores"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var req hrm.BulkScoresRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		if err := validator.New().Struct(req); err != nil {
			var vErrs validator.ValidationErrors
			errors.As(err, &vErrs)
			log.Error("validation failed", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.ValidationErrors(vErrs))
			return
		}

		err := repo.BulkAddCompetencyScores(r.Context(), req)
		if err != nil {
			log.Error("failed to bulk add scores", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to add scores"))
			return
		}

		log.Info("scores bulk added", slog.Int64("assessment_id", req.AssessmentID), slog.Int("count", len(req.Scores)))
		render.Status(r, http.StatusCreated)
		render.JSON(w, r, resp.OK())
	}
}

func EditScore(log *slog.Logger, repo ScoreRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.EditScore"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		var req hrm.EditScoreRequest
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to decode request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid request format"))
			return
		}

		err = repo.EditCompetencyScore(r.Context(), id, req)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("score not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Score not found"))
				return
			}
			log.Error("failed to update score", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to update score"))
			return
		}

		log.Info("score updated", slog.Int64("id", id))
		render.JSON(w, r, resp.OK())
	}
}

func DeleteScore(log *slog.Logger, repo ScoreRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.hrm.competency.DeleteScore"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		idStr := chi.URLParam(r, "id")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Warn("invalid 'id' parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("Invalid 'id' parameter"))
			return
		}

		err = repo.DeleteCompetencyScore(r.Context(), id)
		if err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				log.Warn("score not found", slog.Int64("id", id))
				render.Status(r, http.StatusNotFound)
				render.JSON(w, r, resp.NotFound("Score not found"))
				return
			}
			log.Error("failed to delete score", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to delete score"))
			return
		}

		log.Info("score deleted", slog.Int64("id", id))
		render.Status(r, http.StatusNoContent)
	}
}
