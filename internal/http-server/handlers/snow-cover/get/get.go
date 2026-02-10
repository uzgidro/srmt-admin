package get

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage/repo"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type SnowCoverGetter interface {
	GetSnowCoverByDates(ctx context.Context, dates []string) ([]repo.SnowCoverRow, error)
}

type snowCoverItem struct {
	OrganizationID   int64           `json:"organization_id"`
	OrganizationName string          `json:"organization_name"`
	Cover            *float64        `json:"cover"`
	Zones            json.RawMessage `json:"zones"`
	ResourceDate     *string         `json:"resource_date"`
}

type periodData struct {
	Date  string          `json:"date"`
	Items []snowCoverItem `json:"items"`
}

type getResponse struct {
	Today     periodData `json:"today"`
	Yesterday periodData `json:"yesterday"`
	YearAgo   periodData `json:"year_ago"`
}

func Get(log *slog.Logger, getter SnowCoverGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.snow-cover.get"

		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		dateStr := r.URL.Query().Get("date")

		var baseDate time.Time
		if dateStr == "" {
			baseDate = time.Now()
		} else {
			parsed, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				log.Warn("invalid date format", slog.String("date", dateStr))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("parameter 'date' must be in YYYY-MM-DD format"))
				return
			}
			baseDate = parsed
		}

		today := baseDate.Format("2006-01-02")
		yesterday := baseDate.AddDate(0, 0, -1).Format("2006-01-02")
		yearAgo := baseDate.AddDate(-1, 0, 0).Format("2006-01-02")

		rows, err := getter.GetSnowCoverByDates(r.Context(), []string{today, yesterday, yearAgo})
		if err != nil {
			log.Error("failed to get snow cover data", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to retrieve snow cover data"))
			return
		}

		response := getResponse{
			Today:     periodData{Date: today, Items: []snowCoverItem{}},
			Yesterday: periodData{Date: yesterday, Items: []snowCoverItem{}},
			YearAgo:   periodData{Date: yearAgo, Items: []snowCoverItem{}},
		}

		for _, row := range rows {
			item := snowCoverItem{
				OrganizationID:   row.OrganizationID,
				OrganizationName: row.OrganizationName,
				Cover:            row.Cover,
				Zones:            row.Zones,
				ResourceDate:     row.ResourceDate,
			}

			switch row.Date {
			case today:
				response.Today.Items = append(response.Today.Items, item)
			case yesterday:
				response.Yesterday.Items = append(response.Yesterday.Items, item)
			case yearAgo:
				response.YearAgo.Items = append(response.YearAgo.Items, item)
			}
		}

		render.JSON(w, r, response)
	}
}
