package lexparser

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

// Search creates a proxy handler for lex-parser search endpoint.
// It forwards requests to {baseURL}/search with query parameters:
//   - searchtitle (required): the search query
//   - page (optional): page number, defaults to 1
func Search(log *slog.Logger, client *http.Client, baseURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.lex-parser.Search"

		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// 1. Get query parameters
		searchTitle := r.URL.Query().Get("searchtitle")
		page := r.URL.Query().Get("page")

		// 2. Validate required parameters
		if searchTitle == "" {
			log.Error("missing required parameter 'searchtitle'")
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, resp.BadRequest("missing required 'searchtitle' parameter"))
			return
		}

		// Default page to 1 if not provided
		if page == "" {
			page = "1"
		}

		// 3. Construct the target URL for the external API
		targetURL := fmt.Sprintf("%s/search?searchtitle=%s&page=%s",
			baseURL,
			url.QueryEscape(searchTitle),
			url.QueryEscape(page),
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
			log.Error("failed to execute proxy request to lex-parser service", sl.Err(err))
			render.Status(r, http.StatusBadGateway)
			render.JSON(w, r, resp.BadGateway("lex-parser service is unavailable"))
			return
		}
		defer proxyResp.Body.Close()

		// 5. Proxy the response back to the client
		// Copy headers (like Content-Type) from the target response to our response
		if contentType := proxyResp.Header.Get("Content-Type"); contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}

		// Write the same status code that we received
		w.WriteHeader(proxyResp.StatusCode)

		// Copy the body directly
		io.Copy(w, proxyResp.Body)
	}
}
