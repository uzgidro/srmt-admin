package gesreport

import (
	"encoding/json"
	"testing"
)

func TestUpsertCascadeDailyWeatherRequest_AllAbsent(t *testing.T) {
	const body = `{"organization_id":10,"date":"2026-04-13"}`
	var req UpsertCascadeDailyWeatherRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if req.OrganizationID != 10 {
		t.Errorf("OrganizationID: got %d, want 10", req.OrganizationID)
	}
	if req.Date != "2026-04-13" {
		t.Errorf("Date: got %q, want 2026-04-13", req.Date)
	}
	if req.Temperature.Set {
		t.Errorf("Temperature.Set: got true, want false (field absent)")
	}
	if req.Temperature.Value != nil {
		t.Errorf("Temperature.Value: got %v, want nil", req.Temperature.Value)
	}
	if req.WeatherCondition.Set {
		t.Errorf("WeatherCondition.Set: got true, want false (field absent)")
	}
	if req.WeatherCondition.Value != nil {
		t.Errorf("WeatherCondition.Value: got %v, want nil", req.WeatherCondition.Value)
	}
}

func TestUpsertCascadeDailyWeatherRequest_AllNull(t *testing.T) {
	const body = `{"organization_id":10,"date":"2026-04-13","temperature":null,"weather_condition":null}`
	var req UpsertCascadeDailyWeatherRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !req.Temperature.Set {
		t.Errorf("Temperature.Set: got false, want true (field present with null)")
	}
	if req.Temperature.Value != nil {
		t.Errorf("Temperature.Value: got %v, want nil", req.Temperature.Value)
	}
	if !req.WeatherCondition.Set {
		t.Errorf("WeatherCondition.Set: got false, want true (field present with null)")
	}
	if req.WeatherCondition.Value != nil {
		t.Errorf("WeatherCondition.Value: got %v, want nil", req.WeatherCondition.Value)
	}
}

func TestUpsertCascadeDailyWeatherRequest_AllValues(t *testing.T) {
	const body = `{"organization_id":10,"date":"2026-04-13","temperature":22.5,"weather_condition":"01d"}`
	var req UpsertCascadeDailyWeatherRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !req.Temperature.Set {
		t.Fatal("Temperature.Set: got false, want true")
	}
	if req.Temperature.Value == nil {
		t.Fatal("Temperature.Value: got nil, want pointer to 22.5")
	}
	if *req.Temperature.Value != 22.5 {
		t.Errorf("*Temperature.Value: got %v, want 22.5", *req.Temperature.Value)
	}
	if !req.WeatherCondition.Set {
		t.Fatal("WeatherCondition.Set: got false, want true")
	}
	if req.WeatherCondition.Value == nil {
		t.Fatal("WeatherCondition.Value: got nil, want pointer to \"01d\"")
	}
	if *req.WeatherCondition.Value != "01d" {
		t.Errorf("*WeatherCondition.Value: got %q, want \"01d\"", *req.WeatherCondition.Value)
	}
}

func TestUpsertCascadeDailyWeatherRequest_Mixed(t *testing.T) {
	// Only temperature set, condition absent
	const body = `{"organization_id":10,"date":"2026-04-13","temperature":15.0}`
	var req UpsertCascadeDailyWeatherRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !req.Temperature.Set || req.Temperature.Value == nil || *req.Temperature.Value != 15.0 {
		t.Errorf("Temperature: got Set=%v Value=%v, want Set=true Value=15.0", req.Temperature.Set, req.Temperature.Value)
	}
	if req.WeatherCondition.Set {
		t.Errorf("WeatherCondition.Set: got true, want false (field absent)")
	}
	if req.WeatherCondition.Value != nil {
		t.Errorf("WeatherCondition.Value: got %v, want nil", req.WeatherCondition.Value)
	}
}
