package reservoir

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"srmt-admin/internal/config"
	"srmt-admin/internal/lib/dto"
	"sync"
	"time"
)

// APIResponseItem represents a single item in the reservoir API response
type APIResponseItem struct {
	ID      int     `json:"id"`
	IDWater int     `json:"id_wather"`
	IDUser  int     `json:"id_user"`
	Date    string  `json:"date"`
	Time    int     `json:"time"`
	Weather string  `json:"weather"`
	Level   float64 `json:"level"`
	Size    float64 `json:"size"`
	ToCome  string  `json:"to_come"`
	ToOut   float64 `json:"to_out"`
	Gentle  float64 `json:"gentle"`
}

// APIResponse represents the full API response
type APIResponse struct {
	Items []APIResponseItem `json:"items"`
}

// Fetcher fetches reservoir data from external HTTP endpoints
type Fetcher struct {
	client          *http.Client
	config          *config.ReservoirConfig
	log             *slog.Logger
	reservoirOrgIDs []int64
}

// NewFetcher creates a new reservoir data fetcher
func NewFetcher(cfg *config.ReservoirConfig, log *slog.Logger, reservoirOrgIDs []int64) *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		config:          cfg,
		log:             log,
		reservoirOrgIDs: reservoirOrgIDs,
	}
}

func (f *Fetcher) GetIDs() []int64 {
	return f.reservoirOrgIDs
}

// FetchAll fetches data from all configured sources in parallel and returns a map of organization ID to metrics
func (f *Fetcher) FetchAll(ctx context.Context) (map[int64]*dto.ReservoirMetrics, error) {
	const op = "reservoir.fetcher.FetchAll"

	result := make(map[int64]*dto.ReservoirMetrics)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Fetch data from each source in parallel
	for _, source := range f.config.Sources {
		wg.Add(1)
		go func(src config.ReservoirSource) {
			defer wg.Done()

			data, err := f.fetchSource(ctx, src.APIID)
			if err != nil {
				f.log.Error("failed to fetch reservoir data", slog.String("op", op), slog.Int("api_id", src.APIID), slog.Any("error", err))
				return
			}

			// Calculate metrics from the response
			metrics := f.calculateMetrics(data)

			mu.Lock()
			result[src.OrganizationID] = metrics
			mu.Unlock()
		}(source)
	}

	wg.Wait()
	return result, nil
}

// fetchSource fetches data from a single API endpoint
func (f *Fetcher) fetchSource(ctx context.Context, apiID int) (*APIResponse, error) {
	url := fmt.Sprintf("%s/api/water/daily?id=%d&limit=12", f.config.BaseURL, apiID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	var data APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &data, nil
}

// calculateMetrics calculates current and diff metrics from the API response
// Current is taken from the first element, Diff is calculated as (current - element at time==6)
func (f *Fetcher) calculateMetrics(data *APIResponse) *dto.ReservoirMetrics {
	if data == nil || len(data.Items) == 0 {
		return nil
	}

	metrics := &dto.ReservoirMetrics{}

	// Get current metrics from first element
	first := data.Items[0]
	metrics.Current = f.extractData(&first)

	// Find element with time == 6 for diff calculation
	var time6Element *APIResponseItem
	for i := range data.Items {
		if data.Items[i].Time == 6 {
			time6Element = &data.Items[i]
			break
		}
	}

	// Calculate diff if we found the 6 o'clock element
	if time6Element != nil {
		currentData := f.extractData(&first)
		time6Data := f.extractData(time6Element)

		if currentData != nil && time6Data != nil {
			metrics.Diff = &dto.ReservoirData{}

			// Calculate differences for each field
			if currentData.Income != nil && time6Data.Income != nil {
				diff := *currentData.Income - *time6Data.Income
				metrics.Diff.Income = &diff
			}
			if currentData.Release != nil && time6Data.Release != nil {
				diff := *currentData.Release - *time6Data.Release
				metrics.Diff.Release = &diff
			}
			if currentData.Level != nil && time6Data.Level != nil {
				diff := *currentData.Level - *time6Data.Level
				metrics.Diff.Level = &diff
			}
			if currentData.Volume != nil && time6Data.Volume != nil {
				diff := *currentData.Volume - *time6Data.Volume
				metrics.Diff.Volume = &diff
			}
		}
	}

	return metrics
}

// extractData extracts reservoir data from a single API response item
func (f *Fetcher) extractData(item *APIResponseItem) *dto.ReservoirData {
	if item == nil {
		return nil
	}

	data := &dto.ReservoirData{}

	// Parse ToCome (income) - it's a string in the API response
	if item.ToCome != "" {
		var income float64
		if _, err := fmt.Sscanf(item.ToCome, "%f", &income); err == nil {
			data.Income = &income
		}
	}

	// Release (to_out)
	release := item.ToOut
	data.Release = &release

	// Level
	level := item.Level
	data.Level = &level

	// Volume (size)
	volume := item.Size
	data.Volume = &volume

	return data
}
