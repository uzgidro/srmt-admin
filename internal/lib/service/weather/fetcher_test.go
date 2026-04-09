package weather

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchDaily_Success(t *testing.T) {
	response := `{
		"lat": 41.2995,
		"lon": 69.2401,
		"daily": [
			{
				"dt": 1775718000,
				"temp": {
					"day": 18.64,
					"min": 13.04,
					"max": 21.46,
					"night": 16.12,
					"eve": 20.72,
					"morn": 13.04
				},
				"weather": [
					{
						"id": 500,
						"main": "Rain",
						"description": "небольшой дождь",
						"icon": "10d"
					}
				]
			}
		]
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request parameters
		q := r.URL.Query()
		if q.Get("lat") != "41.300000" {
			t.Errorf("expected lat=41.300000, got %s", q.Get("lat"))
		}
		if q.Get("lon") != "69.240000" {
			t.Errorf("expected lon=69.240000, got %s", q.Get("lon"))
		}
		if q.Get("units") != "metric" {
			t.Errorf("expected units=metric, got %s", q.Get("units"))
		}
		if q.Get("exclude") != "current,minutely,hourly,alerts" {
			t.Errorf("expected exclude=current,minutely,hourly,alerts, got %s", q.Get("exclude"))
		}
		if q.Get("lang") != "ru" {
			t.Errorf("expected lang=ru, got %s", q.Get("lang"))
		}
		if q.Get("appid") != "test-key" {
			t.Errorf("expected appid=test-key, got %s", q.Get("appid"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	}))
	defer srv.Close()

	f := NewFetcher(srv.Client(), srv.URL, "test-key")

	data, err := f.FetchDaily(context.Background(), 41.3, 69.24)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if data.Temperature != 18.64 {
		t.Errorf("expected temperature 18.64, got %f", data.Temperature)
	}
	if data.Icon != "10d" {
		t.Errorf("expected icon '10d', got %q", data.Icon)
	}
}

func TestFetchDaily_EmptyDaily(t *testing.T) {
	response := `{"daily": []}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	}))
	defer srv.Close()

	f := NewFetcher(srv.Client(), srv.URL, "test-key")

	_, err := f.FetchDaily(context.Background(), 41.3, 69.24)
	if err == nil {
		t.Fatal("expected error for empty daily array")
	}
}

func TestFetchDaily_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"cod": 401, "message": "Invalid API key"}`))
	}))
	defer srv.Close()

	f := NewFetcher(srv.Client(), srv.URL, "bad-key")

	_, err := f.FetchDaily(context.Background(), 41.3, 69.24)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestFetchDaily_EmptyWeather(t *testing.T) {
	response := `{
		"daily": [
			{
				"temp": {"day": 20.5},
				"weather": []
			}
		]
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(response))
	}))
	defer srv.Close()

	f := NewFetcher(srv.Client(), srv.URL, "test-key")

	_, err := f.FetchDaily(context.Background(), 41.3, 69.24)
	if err == nil {
		t.Fatal("expected error for empty weather array")
	}
}
