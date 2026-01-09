package investments

import (
	"context"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/dto"
	"srmt-admin/internal/lib/helpers"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/investment"
	"strconv"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type investmentGetter interface {
	GetAllInvestments(ctx context.Context, filters dto.GetAllInvestmentsFilters) ([]*investment.ResponseModel, error)
}

func GetAll(log *slog.Logger, getter investmentGetter, minioRepo helpers.MinioURLGenerator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.investment.get-all"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		// Parse query parameters for filtering
		filters := dto.GetAllInvestmentsFilters{}
		q := r.URL.Query()

		if typeIDStr := q.Get("type_id"); typeIDStr != "" {
			typeID, err := strconv.Atoi(typeIDStr)
			if err != nil {
				log.Warn("invalid 'type_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'type_id' parameter"))
				return
			}
			filters.TypeID = &typeID
		}

		if statusIDStr := q.Get("status_id"); statusIDStr != "" {
			statusID, err := strconv.Atoi(statusIDStr)
			if err != nil {
				log.Warn("invalid 'status_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'status_id' parameter"))
				return
			}
			filters.StatusID = &statusID
		}

		if minCostStr := q.Get("min_cost"); minCostStr != "" {
			minCost, err := strconv.ParseFloat(minCostStr, 64)
			if err != nil {
				log.Warn("invalid 'min_cost' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'min_cost' parameter"))
				return
			}
			filters.MinCost = &minCost
		}

		if maxCostStr := q.Get("max_cost"); maxCostStr != "" {
			maxCost, err := strconv.ParseFloat(maxCostStr, 64)
			if err != nil {
				log.Warn("invalid 'max_cost' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'max_cost' parameter"))
				return
			}
			filters.MaxCost = &maxCost
		}

		if nameSearch := q.Get("name_search"); nameSearch != "" {
			filters.NameSearch = &nameSearch
		}

		if createdByStr := q.Get("created_by_user_id"); createdByStr != "" {
			createdByID, err := strconv.ParseInt(createdByStr, 10, 64)
			if err != nil {
				log.Warn("invalid 'created_by_user_id' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'created_by_user_id' parameter"))
				return
			}
			filters.CreatedByUserID = &createdByID
		}

		investments, err := getter.GetAllInvestments(r.Context(), filters)
		if err != nil {
			log.Error("failed to get all investments", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve investments"))
			return
		}

		// Transform investments to include presigned URLs
		investmentsWithURLs := make([]*investment.ResponseWithURLs, 0, len(investments))
		for _, inv := range investments {
			invWithURLs := &investment.ResponseWithURLs{
				ID:            inv.ID,
				Name:          inv.Name,
				Type:          inv.Type,
				Status:        inv.Status,
				Cost:          inv.Cost,
				Comments:      inv.Comments,
				CreatedAt:     inv.CreatedAt,
				CreatedByUser: inv.CreatedByUser,
				UpdatedAt:     inv.UpdatedAt,
				Files:         helpers.TransformFilesWithURLs(r.Context(), inv.Files, minioRepo, log),
			}
			investmentsWithURLs = append(investmentsWithURLs, invWithURLs)
		}

		log.Info("successfully retrieved investments", slog.Int("count", len(investmentsWithURLs)))
		render.JSON(w, r, investmentsWithURLs)
	}
}
