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

		// Get page parameter (default = 1)
		page := r.URL.Query().Get("page")
		if page == "" {
			page = "1"
		}

		// Construct the target URL
		targetURL := fmt.Sprintf("%s/api/news/page/%s", baseURL, page)

		// Create proxy request
		proxyReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, targetURL, nil)
		if err != nil {
			log.Error("failed to create proxy request", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("internal error"))
			return
		}

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
		if contentType := proxyResp.Header.Get("Content-Type"); contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}

		w.WriteHeader(proxyResp.StatusCode)

		io.Copy(w, proxyResp.Body)
	}
}
