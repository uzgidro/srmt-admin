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
func (f *Fetcher) FetchAll(ctx context.Context, date string) (map[int64]*dto.ReservoirMetrics, error) {
	const op = "reservoir.fetcher.FetchAll"

	result := make(map[int64]*dto.ReservoirMetrics)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Check if the requested date is today
	today := time.Now().Format("2006-01-02")
	isToday := date == today

	// Fetch data from each source in parallel
	for _, source := range f.config.Sources {
		wg.Add(1)
		go func(src config.ReservoirSource) {
			defer wg.Done()

			data, err := f.fetchSource(ctx, src.APIID, date)
			if err != nil {
				f.log.Error("failed to fetch reservoir data", slog.String("op", op), slog.Int("api_id", src.APIID), slog.Any("error", err))
				return
			}

			// Calculate metrics from the response
			metrics := f.calculateMetrics(data, isToday)

			mu.Lock()
			result[src.OrganizationID] = metrics
			mu.Unlock()
		}(source)
	}

	wg.Wait()
	return result, nil
}

// FetchDataAtDayBegin - Data at 06:00
func (f *Fetcher) FetchDataAtDayBegin(ctx context.Context, date string) (map[int64]*dto.OrganizationWithData, error) {
	const op = "reservoir.fetcher.FetchAll"

	result := make(map[int64]*dto.OrganizationWithData)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Fetch data from each source in parallel
	for _, source := range f.config.Sources {
		wg.Add(1)
		go func(src config.ReservoirSource) {
			defer wg.Done()

			rawData, err := f.fetchSource(ctx, src.APIID, date)
			if err != nil {
				f.log.Error("failed to fetch reservoir data", slog.String("op", op), slog.Int("api_id", src.APIID), slog.Any("error", err))
				return
			}

			convertedData := f.convertRawData(rawData)

			dataAtDayBegin := &dto.ReservoirData{}
			for _, item := range convertedData {
				if item.Time.Hour() == 6 {
					dataAtDayBegin = item
				}
			}

			finalData := &dto.OrganizationWithData{
				OrganizationID: src.OrganizationID,
				ReservoirAPIID: int64(src.APIID),
				Data:           dataAtDayBegin,
			}

			mu.Lock()
			result[finalData.OrganizationID] = finalData
			mu.Unlock()
		}(source)
	}

	wg.Wait()
	return result, nil
}

// FetchHourly fetches hourly reservoir data from all configured sources and returns the last 6 hourly data points for each
func (f *Fetcher) FetchHourly(ctx context.Context, date string) (map[int64][]*dto.ReservoirData, error) {
	const op = "reservoir.fetcher.FetchHourly"

	result := make(map[int64][]*dto.ReservoirData)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Fetch data from each source in parallel
	for _, source := range f.config.Sources {
		wg.Add(1)
		go func(src config.ReservoirSource) {
			defer wg.Done()

			// Fetch from range endpoint
			data, err := f.fetchSourceRange(ctx, src.APIID, date)
			if err != nil {
				f.log.Error("failed to fetch hourly reservoir data",
					slog.String("op", op),
					slog.Int("api_id", src.APIID),
					slog.Any("error", err))
				return
			}

			// Filter and convert to hourly data
			hourlyData := f.filterHourlyData(data, date)

			mu.Lock()
			result[src.OrganizationID] = hourlyData
			mu.Unlock()
		}(source)
	}

	wg.Wait()
	return result, nil
}

// fetchSource fetches data from a single API endpoint
func (f *Fetcher) fetchSource(ctx context.Context, apiID int, date string) (*APIResponse, error) {
	url := fmt.Sprintf("%s/api/water/water-date?id=%d&date=%s", f.config.BaseURL, apiID, date)

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

// fetchSourceRange fetches data from a single API endpoint using date range
func (f *Fetcher) fetchSourceRange(ctx context.Context, apiID int, date string) (*APIResponse, error) {
	// Parse the given date in local timezone
	givenDate, err := time.ParseInLocation("2006-01-02", date, time.Local)
	if err != nil {
		return nil, fmt.Errorf("failed to parse date: %w", err)
	}

	// Calculate date_from (day before) and date_to (day after)
	dateFrom := givenDate.AddDate(0, 0, -1).Format("2006-01-02")
	dateTo := givenDate.AddDate(0, 0, 1).Format("2006-01-02")

	url := fmt.Sprintf("%s/api/water/water-range?id=%d&date_from=%s&date_to=%s",
		f.config.BaseURL, apiID, dateFrom, dateTo)

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

func (f *Fetcher) convertRawData(data *APIResponse) []*dto.ReservoirData {
	result := make([]*dto.ReservoirData, 0)
	for _, res := range data.Items {
		result = append(result, f.extractDataWithTimestamp(&res))
	}

	return result
}

// filterHourlyData filters API response data and returns the last 6 hourly data points
func (f *Fetcher) filterHourlyData(data *APIResponse, givenDate string) []*dto.ReservoirData {
	if data == nil || len(data.Items) == 0 {
		return []*dto.ReservoirData{}
	}

	// Parse the given date in local timezone
	parsedDate, err := time.ParseInLocation("2006-01-02", givenDate, time.Local)
	if err != nil {
		f.log.Error("failed to parse given date", slog.String("date", givenDate), slog.Any("error", err))
		return []*dto.ReservoirData{}
	}

	// Calculate cutoff time: given_date + 1 day at 00:00
	cutoffTime := parsedDate.AddDate(0, 0, 1)

	// Convert API items to timestamped data and find cutoff index
	cutoffIndex := -1
	for i, item := range data.Items {
		// Parse item date in local timezone
		itemDate, err := time.ParseInLocation("2006-01-02", item.Date, time.Local)
		if err != nil {
			f.log.Error("failed to parse item date", slog.String("date", item.Date), slog.Any("error", err))
			continue
		}

		// Combine date with time (hours)
		itemTimestamp := time.Date(
			itemDate.Year(), itemDate.Month(), itemDate.Day(),
			item.Time, 0, 0, 0, time.Local,
		)

		// Check if this is the cutoff point
		if itemTimestamp.Equal(cutoffTime) {
			cutoffIndex = i
			break
		}
	}

	// Slice the items if cutoff was found
	items := data.Items
	if cutoffIndex != -1 {
		items = items[:cutoffIndex+1]
	}

	// Filter to keep only items where hour % 6 == 0 (0, 6, 12, 18)
	var filteredItems []APIResponseItem
	for _, item := range items {
		if item.Time%6 == 0 {
			filteredItems = append(filteredItems, item)
		}
	}

	// Take the last 6 elements (or fewer if not enough data)
	startIndex := 0
	if len(filteredItems) > 6 {
		startIndex = len(filteredItems) - 6
	}
	lastSixItems := filteredItems[startIndex:]

	// Convert to ReservoirData array
	result := make([]*dto.ReservoirData, 0, len(lastSixItems))
	for i := range lastSixItems {
		item := &lastSixItems[i]
		reservoirData := f.extractDataWithTimestamp(item)
		if reservoirData != nil {
			result = append(result, reservoirData)
		}
	}

	return result
}

// extractDataWithTimestamp extracts reservoir data with timestamp and weather from an API response item
func (f *Fetcher) extractDataWithTimestamp(item *APIResponseItem) *dto.ReservoirData {
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

	reservoirAPIID := int64(item.IDWater)
	data.ReservoirAPIID = &reservoirAPIID

	// Release (to_out)
	release := item.ToOut
	data.Release = &release

	// Level
	level := item.Level
	data.Level = &level

	// Volume (size)
	volume := item.Size
	data.Volume = &volume

	// Parse date in local timezone and combine with time to create timestamp
	itemDate, err := time.ParseInLocation("2006-01-02", item.Date, time.Local)
	if err == nil {
		timestamp := time.Date(
			itemDate.Year(), itemDate.Month(), itemDate.Day(),
			item.Time, 0, 0, 0, time.Local,
		)
		data.Time = &timestamp
	}

	// Weather
	if item.Weather != "" {
		weather := item.Weather
		data.Weather = &weather
	}

	return data
}

// calculateMetrics calculates current and diff metrics from the API response
// For today: Current is taken from the last (newest) element, Diff is calculated as (current - element at time==6)
// For historical dates: Current is taken from element with time==6 (or last if 6 doesn't exist), Diff is not calculated
// Returns ReservoirMetrics with 0 values if data is empty or nil
func (f *Fetcher) calculateMetrics(data *APIResponse, isToday bool) *dto.ReservoirMetrics {
	if data == nil || len(data.Items) == 0 {
		// Return metrics with 0 values
		zero := 0.0
		return &dto.ReservoirMetrics{
			Current: &dto.ReservoirData{
				Income:  &zero,
				Release: &zero,
				Level:   &zero,
				Volume:  &zero,
			},
			Diff: &dto.ReservoirData{
				Income:  &zero,
				Release: &zero,
				Level:   &zero,
				Volume:  &zero,
			},
		}
	}

	metrics := &dto.ReservoirMetrics{}

	if isToday {
		// For today: use the last (newest) element for current
		last := data.Items[len(data.Items)-1]
		metrics.Current = f.extractData(&last)

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
			currentData := f.extractData(&last)
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
	} else {
		// For historical dates: find element with time == 6, or use last element if not found
		var selectedElement *APIResponseItem
		for i := range data.Items {
			if data.Items[i].Time == 6 {
				selectedElement = &data.Items[i]
				break
			}
		}

		// If no time==6 element found, use the last element
		if selectedElement == nil {
			selectedElement = &data.Items[len(data.Items)-1]
		}

		metrics.Current = f.extractData(selectedElement)
		// No diff calculation for historical dates
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
