package proxy

import (
	"fmt"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"io"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"
	"strconv"
)

func New(log *slog.Logger, client *http.Client, baseURL, apiKey, path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.weather.proxy.New"

		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// 1. Get lat and lon from query parameters
		latStr := r.URL.Query().Get("lat")
		lonStr := r.URL.Query().Get("lon")

		// 2. Validate parameters
		if _, err := strconv.ParseFloat(latStr, 64); err != nil {
			log.Error("invalid latitude parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid or missing 'lat' parameter"))
			return
		}
		if _, err := strconv.ParseFloat(lonStr, 64); err != nil {
			log.Error("invalid longitude parameter", sl.Err(err))
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("invalid or missing 'lon' parameter"))
			return
		}

		// 3. Construct the target URL for the external API
		targetURL := fmt.Sprintf("%s%s?lat=%s&lon=%s&appid=%s&units=metric",
			baseURL,
			path,
			latStr,
			lonStr,
			apiKey,
		)

		// 4. Create and execute the proxy request
		proxyReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, targetURL, nil)
		if err != nil {
			log.Error("failed to create proxy request", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("internal error"))
			return
		}

		proxyResp, err := client.Do(proxyReq)
		if err != nil {
			log.Error("failed to execute proxy request to weather service", sl.Err(err))
			render.Status(r, http.StatusBadGateway)
			render.JSON(w, r, resp.BadGateway("weather service is unavailable"))
			return
		}
		defer proxyResp.Body.Close()

		// 5. Proxy the response back to the client
		// Copy headers (like Content-Type) from the target response to our response
		for key, values := range proxyResp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		// Write the same status code that we received
		w.WriteHeader(proxyResp.StatusCode)

		// Copy the body (the JSON) directly
		io.Copy(w, proxyResp.Body)
	}
}
