package currency

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	resp "srmt-admin/internal/lib/api/response"
	"srmt-admin/internal/lib/logger/sl"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

const cbuURL = "https://cbu.uz/ru/arkhiv-kursov-valyut/json/"

// CurrencyItem represents a single currency from CBU API
type CurrencyItem struct {
	Ccy  string      `json:"Ccy"`
	Rate interface{} `json:"Rate"` // Can be string, number, or undefined
}

// Response represents the handler response
type Response struct {
	resp.Response
	Rate float64 `json:"rate"`
}

func Get(log *slog.Logger, httpClient *http.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.currency.Get"

		log = log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		// Create request with timeout context
		ctx, cancel := r.Context(), func() {}
		if httpClient.Timeout == 0 {
			var ctxCancel func()
			ctx, ctxCancel = r.Context(), ctxCancel
			cancel = ctxCancel
		}
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, cbuURL, nil)
		if err != nil {
			log.Error("failed to create request", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to create request"))
			return
		}

		// Make request to CBU API
		httpResp, err := httpClient.Do(req)
		if err != nil {
			log.Error("failed to fetch currency data from CBU", sl.Err(err))
			render.Status(r, http.StatusBadGateway)
			render.JSON(w, r, resp.BadGateway("Failed to fetch currency data from external API"))
			return
		}
		defer httpResp.Body.Close()

		if httpResp.StatusCode != http.StatusOK {
			log.Error("CBU API returned non-200 status", slog.Int("status", httpResp.StatusCode))
			render.Status(r, http.StatusBadGateway)
			render.JSON(w, r, resp.BadGateway("External API returned error"))
			return
		}

		// Read response body
		body, err := io.ReadAll(httpResp.Body)
		if err != nil {
			log.Error("failed to read response body", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to read response"))
			return
		}

		// Parse JSON
		var currencies []CurrencyItem
		if err := json.Unmarshal(body, &currencies); err != nil {
			log.Error("failed to parse JSON", sl.Err(err))
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, resp.InternalServerError("Failed to parse response"))
			return
		}

		// Find USD currency
		var usdRate float64
		var found bool

		for _, curr := range currencies {
			if curr.Ccy == "USD" {
				usdRate, err = parseRate(curr.Rate)
				if err != nil {
					log.Error("failed to parse USD rate", sl.Err(err), slog.Any("rate_value", curr.Rate))
					render.Status(r, http.StatusInternalServerError)
					render.JSON(w, r, resp.InternalServerError("Failed to parse USD rate"))
					return
				}
				found = true
				break
			}
		}

		if !found {
			log.Warn("USD currency not found in CBU response")
			render.Status(r, http.StatusNotFound)
			render.JSON(w, r, resp.NotFound("USD currency not found"))
			return
		}

		log.Info("successfully fetched USD rate", slog.Float64("rate", usdRate))
		render.JSON(w, r, Response{
			Response: resp.OK(),
			Rate:     usdRate,
		})
	}
}

// parseRate parses the rate field which can be string, number, or undefined
func parseRate(rateValue interface{}) (float64, error) {
	if rateValue == nil {
		return 0, fmt.Errorf("rate is undefined")
	}

	switch v := rateValue.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case string:
		rate, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("failed to parse rate string: %w", err)
		}
		return rate, nil
	default:
		return 0, fmt.Errorf("unsupported rate type: %T", v)
	}
}
