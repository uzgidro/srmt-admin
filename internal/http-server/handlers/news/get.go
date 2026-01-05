package news

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

func New(log *slog.Logger, client *http.Client, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.news.New"

		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// Get query parameters
		limit := r.URL.Query().Get("limit")
		offsetID := r.URL.Query().Get("offset_id")
		dateFrom := r.URL.Query().Get("date_from")
		dateTo := r.URL.Query().Get("date_to")

		// Construct the target URL
		targetURL := fmt.Sprintf("%s/api/v1/messages", baseURL)

		// Create proxy request
		proxyReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, targetURL, nil)
		if err != nil {
			log.Error("failed to create proxy request", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("internal error"))
			return
		}

		// Add query parameters to proxy request
		q := proxyReq.URL.Query()
		if limit != "" {
			q.Add("limit", limit)
		}
		if offsetID != "" {
			q.Add("offset_id", offsetID)
		}
		if dateFrom != "" {
			q.Add("date_from", dateFrom)
		}
		if dateTo != "" {
			q.Add("date_to", dateTo)
		}
		proxyReq.URL.RawQuery = q.Encode()

		// Execute the proxy request
		proxyResp, err := client.Do(proxyReq)
		if err != nil {
			log.Error("failed to execute proxy request to news-retriever service", sl.Err(err))
			render.Status(r, http.StatusBadGateway)
			render.JSON(w, r, resp.BadGateway("news-retriever service is unavailable"))
			return
		}
		defer proxyResp.Body.Close()

		// Proxy the response back to the client
		// Copy headers (like Content-Type) from the target response to our response
		if contentType := proxyResp.Header.Get("Content-Type"); contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}

		// Write the same status code that we received
		w.WriteHeader(proxyResp.StatusCode)

		// Copy the body (the JSON) directly
		io.Copy(w, proxyResp.Body)
	}
}
