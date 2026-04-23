package shutdowns

import (
	"context"
	"log/slog"
	"net/http"
	mwauth "srmt-admin/internal/http-server/middleware/auth"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/helpers"
	"srmt-admin/internal/lib/logger/sl"
	"srmt-admin/internal/lib/model/shutdown"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

type shutdownGetter interface {
	GetShutdowns(ctx context.Context, day time.Time) ([]*shutdown.ResponseModel, error)
	GetShutdownsByCascade(ctx context.Context, day time.Time, cascadeOrgID int64) ([]*shutdown.ResponseModel, error)
	GetOrganizationTypesMap(ctx context.Context) (map[int64][]string, error)
}

const layout = "2006-01-02" // YYYY-MM-DD

// fetchShutdownsForCaller returns shutdowns visible to the caller for the
// given day plus an audit-log scope label. sc/rais and roles other than
// "cascade" see everything. A "cascade" caller sees only shutdowns of
// their own cascade and its direct stations. A cascade caller without
// an OrganizationID sees an empty list (no leak, no error).
func fetchShutdownsForCaller(
	ctx context.Context,
	getter shutdownGetter,
	day time.Time,
) ([]*shutdown.ResponseModel, string, int64, error) {
	claims, ok := mwauth.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		list, err := getter.GetShutdowns(ctx, day)
		return list, "all", 0, err
	}

	// sc/rais — full access (early return).
	for _, role := range claims.Roles {
		if role == "sc" || role == "rais" {
			list, err := getter.GetShutdowns(ctx, day)
			return list, "all", claims.UserID, err
		}
	}

	// cascade — restricted to own cascade.
	for _, role := range claims.Roles {
		if role == "cascade" {
			if claims.OrganizationID == 0 {
				return []*shutdown.ResponseModel{}, "empty-no-org", claims.UserID, nil
			}
			list, err := getter.GetShutdownsByCascade(ctx, day, claims.OrganizationID)
			return list, "cascade", claims.UserID, err
		}
	}

	// Other roles unchanged: see all.
	list, err := getter.GetShutdowns(ctx, day)
	return list, "all", claims.UserID, err
}

func Get(log *slog.Logger, getter shutdownGetter, minioRepo helpers.MinioURLGenerator, loc *time.Location) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.shutdown.Get"
		log := log.With(slog.String("op", op), slog.String("request_id", middleware.GetReqID(r.Context())))

		var day time.Time
		dateStr := r.URL.Query().Get("date")

		if dateStr == "" {
			now := time.Now().In(loc)
			// День начинается в 05:00 местного времени
			day = time.Date(now.Year(), now.Month(), now.Day(), 5, 0, 0, 0, loc)
		} else {
			var err error
			// Parse the date in the configured timezone
			t, err := time.ParseInLocation(layout, dateStr, loc)
			if err != nil {
				log.Warn("invalid 'date' parameter", sl.Err(err))
				render.Status(r, http.StatusBadRequest)
				render.JSON(w, r, resp.BadRequest("Invalid 'date' format, use YYYY-MM-DD"))
				return
			}
			// День начинается в 05:00 местного времени
			day = time.Date(t.Year(), t.Month(), t.Day(), 5, 0, 0, 0, loc)
		}

		shutdowns, scope, userID, err := fetchShutdownsForCaller(r.Context(), getter, day)
		if err != nil {
			log.Error("failed to get all shutdowns", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve shutdowns"))
			return
		}

		orgTypesMap, err := getter.GetOrganizationTypesMap(r.Context())
		if err != nil {
			log.Error("failed to get organization types", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to retrieve organization types"))
			return
		}

		// Transform function to convert shutdown to WithURLs version
		transformShutdown := func(s *shutdown.ResponseModel) *shutdown.ResponseWithURLs {
			return &shutdown.ResponseWithURLs{
				ID:                            s.ID,
				OrganizationID:                s.OrganizationID,
				OrganizationName:              s.OrganizationName,
				StartedAt:                     s.StartedAt,
				EndedAt:                       s.EndedAt,
				Reason:                        s.Reason,
				CreatedByUser:                 s.CreatedByUser,
				GenerationLossMwh:             s.GenerationLossMwh,
				CreatedAt:                     s.CreatedAt,
				Viewed:                        s.Viewed,
				IdleDischargeVolumeThousandM3: s.IdleDischargeVolumeThousandM3,
				Files:                         helpers.TransformFilesWithURLs(r.Context(), s.Files, minioRepo, log),
			}
		}

		response := shutdown.GroupedResponseWithURLs{
			Ges:   make([]*shutdown.ResponseWithURLs, 0),
			Mini:  make([]*shutdown.ResponseWithURLs, 0),
			Micro: make([]*shutdown.ResponseWithURLs, 0),
			Other: make([]*shutdown.ResponseWithURLs, 0),
		}

		for _, s := range shutdowns {
			types, ok := orgTypesMap[s.OrganizationID]
			if !ok {
				response.Other = append(response.Other, transformShutdown(s))
				continue
			}

			wasGrouped := false
			for _, t := range types {
				switch t {
				case "ges":
					response.Ges = append(response.Ges, transformShutdown(s))
					wasGrouped = true
				case "mini":
					response.Mini = append(response.Mini, transformShutdown(s))
					wasGrouped = true
				case "micro":
					response.Micro = append(response.Micro, transformShutdown(s))
					wasGrouped = true
				}
			}
			if !wasGrouped {
				response.Other = append(response.Other, transformShutdown(s))
			}
		}

		log.Info("retrieved shutdowns",
			slog.Int("count", len(shutdowns)),
			slog.String("scope", scope),
			slog.Int64("user_id", userID),
		)
		render.JSON(w, r, response)
	}
}
