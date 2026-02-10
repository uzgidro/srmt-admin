package snowcover

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"time"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/storage/repo"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type SnowCoverUpserter interface {
	UpsertSnowCoverBatch(ctx context.Context, date string, resourceDate string, items []repo.SnowCoverItem) error
}

type Request struct {
	Date       string      `json:"date"`
	Catchments []Catchment `json:"catchments"`
}

type Zone struct {
	MinElev int      `json:"min_elev"`
	MaxElev int      `json:"max_elev"`
	ScaPct  *float64 `json:"sca_pct"`
}

type Catchment struct {
	Name   string   `json:"name"`
	ScaPct *float64 `json:"sca_pct"`
	Zones  []Zone   `json:"zones"`
}

// catchment name → organization_id
var catchmentOrgMap = map[string]int64{
	"Chirchik":            100,
	"Ahangaran_Irtash":    97,
	"piskem_mullala":      103,
	"Tupalang_zarchob":    99,
	"Chatkal_Hudaydodsay": 104,
	"Karadaryo_Andijan":   96,
	"Akdarya_Gissarak":    98,
}

func New(log *slog.Logger, upserter SnowCoverUpserter, loc *time.Location) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.snow-cover.post"

		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request
		if err := render.DecodeJSON(r.Body, &req); err != nil {
			log.Error("failed to parse request", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("failed to parse request body"))
			return
		}

		if len(req.Catchments) == 0 {
			log.Warn("empty catchments array")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("field 'catchments' must contain at least one item"))
			return
		}

		var resourceDate string
		if req.Date != "" {
			if !regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`).MatchString(req.Date) {
				log.Warn("invalid date format", slog.String("date", req.Date))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("field 'date' must be in YYYY-MM-DD format"))
				return
			}
			resourceDate = req.Date
		}

		date := time.Now().In(loc).Format("2006-01-02")

		var items []repo.SnowCoverItem
		for _, c := range req.Catchments {
			orgID, ok := catchmentOrgMap[c.Name]
			if !ok {
				continue
			}

			var zones json.RawMessage
			if len(c.Zones) > 0 {
				z, err := json.Marshal(c.Zones)
				if err != nil {
					log.Error("failed to marshal zones", sl.Err(err), slog.String("catchment", c.Name))
					render.Status(r, http.StatusInternalServerError)
					render.JSON(w, r, resp.InternalServerError("failed to process zones data"))
					return
				}
				zones = z
			}

			items = append(items, repo.SnowCoverItem{
				OrganizationID: orgID,
				Cover:          c.ScaPct,
				Zones:          zones,
			})
		}

		if len(items) == 0 {
			log.Info("no mapped catchments found, nothing to save")
			render.Status(r, http.StatusOK)
			render.JSON(w, r, map[string]string{"status": "ok"})
			return
		}

		if err := upserter.UpsertSnowCoverBatch(r.Context(), date, resourceDate, items); err != nil {
			log.Error("failed to upsert snow cover", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("failed to save snow cover data"))
			return
		}

		log.Info("snow cover saved", slog.String("date", date), slog.Int("count", len(items)))

		render.Status(r, http.StatusOK)
		render.JSON(w, r, map[string]string{"status": "ok"})
	}
}
