package ascue

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"srmt-admin/internal/config"
	"srmt-admin/internal/lib/dto"
	"strconv"
	"strings"
	"sync"
	"time"
)

// FlexibleFloat handles JSON unmarshaling of both string and float values
type FlexibleFloat float64

// UnmarshalJSON implements custom unmarshaling to handle both string and float values
func (f *FlexibleFloat) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as float first
	var floatVal float64
	if err := json.Unmarshal(data, &floatVal); err == nil {
		*f = FlexibleFloat(floatVal)
		return nil
	}

	// If that fails, try to unmarshal as string and convert
	var strVal string
	if err := json.Unmarshal(data, &strVal); err != nil {
		return fmt.Errorf("value is neither float nor string: %w", err)
	}

	// Convert string to float (trim whitespace first)
	floatVal, err := strconv.ParseFloat(strings.TrimSpace(strVal), 64)
	if err != nil {
		return fmt.Errorf("failed to parse string value to float: %w", err)
	}

	*f = FlexibleFloat(floatVal)
	return nil
}

// APIResponseItem represents a single item in the ASCUE API response array
type APIResponseItem struct {
	ID    int           `json:"id"`
	Value FlexibleFloat `json:"value"`
}

// Fetcher fetches ASCUE data from external HTTP endpoints
type Fetcher struct {
	client *http.Client
	config *config.ASCUEConfig
	log    *slog.Logger
}

// NewFetcher creates a new ASCUE data fetcher
func NewFetcher(cfg *config.ASCUEConfig, log *slog.Logger) *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		config: cfg,
		log:    log,
	}
}

// FetchAll fetches data from all configured sources in parallel and returns a map of organization ID to metrics
func (f *Fetcher) FetchAll(ctx context.Context) (map[int64]*dto.ASCUEMetrics, error) {
	const op = "ascue.fetcher.FetchAll"

	result := make(map[int64]*dto.ASCUEMetrics)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Fetch data from each source in parallel
	for _, source := range f.config.Sources {
		wg.Add(1)
		go func(src config.Source) {
			defer wg.Done()

			data, err := f.fetchSource(ctx, src.URL)
			if err != nil {
				f.log.Error("failed to fetch ASCUE data", slog.String("op", op), slog.String("url", src.URL), slog.Any("error", err))
				return
			}

			// Parse metrics for the cascade organization
			metrics := f.parseMetrics(data, src.Metrics, src.Aggregates)

			mu.Lock()
			result[src.OrganizationID] = metrics
			mu.Unlock()

			// Parse metrics for child organizations
			for _, child := range src.Children {
				childMetrics := f.parseMetrics(data, child.Metrics, child.Aggregates)
				mu.Lock()
				result[child.OrganizationID] = childMetrics
				mu.Unlock()
			}
		}(source)
	}

	wg.Wait()
	return result, nil
}

// fetchSource fetches data from a single HTTP endpoint
func (f *Fetcher) fetchSource(ctx context.Context, url string) ([]APIResponseItem, error) {
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

	var data []APIResponseItem
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return data, nil
}

// parseMetrics extracts metrics from the API response array using the provided mappings
func (f *Fetcher) parseMetrics(data []APIResponseItem, metrics config.MetricMapping, aggregates config.AggregateMapping) *dto.ASCUEMetrics {
	result := &dto.ASCUEMetrics{}

	// Create a map for quick lookup by ID
	dataMap := make(map[int]float64)
	for _, item := range data {
		dataMap[item.ID] = float64(item.Value)
	}

	// Extract metric values
	if val, ok := dataMap[metrics.Active]; ok {
		result.Active = &val
	}
	if val, ok := dataMap[metrics.Reactive]; ok {
		result.Reactive = &val
	}
	if metrics.PowerImport != nil {
		if val, ok := dataMap[*metrics.PowerImport]; ok {
			result.PowerImport = &val
		}
	}
	if metrics.PowerExport != nil {
		if val, ok := dataMap[*metrics.PowerExport]; ok {
			result.PowerExport = &val
		}
	}
	if metrics.OwnNeeds != nil {
		if val, ok := dataMap[*metrics.OwnNeeds]; ok {
			result.OwnNeeds = &val
		}
	}
	if metrics.Flow != nil {
		if val, ok := dataMap[*metrics.Flow]; ok {
			result.Flow = &val
		}
	}

	// Extract aggregate counts (convert float to int)
	if val, ok := dataMap[aggregates.Active]; ok {
		intVal := int(val)
		result.ActiveAggCount = &intVal
	}
	if val, ok := dataMap[aggregates.Pending]; ok {
		intVal := int(val)
		result.PendingAggCount = &intVal
	}
	if val, ok := dataMap[aggregates.Repair]; ok {
		intVal := int(val)
		result.RepairAggCount = &intVal
	}

	return result
}
