package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type WeatherData struct {
	Temperature float64
	Icon        string // icon code, e.g. "10d"
}

type Fetcher struct {
	client  *http.Client
	baseURL string
	apiKey  string
}

func NewFetcher(client *http.Client, baseURL, apiKey string) *Fetcher {
	return &Fetcher{
		client:  client,
		baseURL: baseURL,
		apiKey:  apiKey,
	}
}

// FetchDaily fetches today's weather forecast for the given coordinates.
func (f *Fetcher) FetchDaily(ctx context.Context, lat, lon float64) (*WeatherData, error) {
	url := fmt.Sprintf("%s/data/3.0/onecall?lat=%f&lon=%f&appid=%s&units=metric&exclude=current,minutely,hourly,alerts&lang=ru",
		f.baseURL, lat, lon, f.apiKey)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("weather: create request: %w", err)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("weather: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weather: API returned %d", resp.StatusCode)
	}

	var result oneCallResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return nil, fmt.Errorf("weather: decode response: %w", err)
	}

	if len(result.Daily) == 0 {
		return nil, fmt.Errorf("weather: empty daily array")
	}

	day := result.Daily[0]
	if len(day.Weather) == 0 {
		return nil, fmt.Errorf("weather: empty weather array in daily[0]")
	}

	return &WeatherData{
		Temperature: day.Temp.Day,
		Icon:        day.Weather[0].Icon,
	}, nil
}

// oneCallResponse is a minimal representation of the OpenWeatherMap One Call 3.0 response.
type oneCallResponse struct {
	Daily []dailyForecast `json:"daily"`
}

type dailyForecast struct {
	Temp    dailyTemp        `json:"temp"`
	Weather []weatherElement `json:"weather"`
}

type dailyTemp struct {
	Day float64 `json:"day"`
}

type weatherElement struct {
	Icon string `json:"icon"`
}
