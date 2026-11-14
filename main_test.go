package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestWeatherFields(t *testing.T) {
	tests := []struct {
		name string
		json string
		want map[string]interface{}
	}{
		{
			name: "positive",
			json: `{"main":{"temp":7.97,"pressure":1011,"humidity":85}}`,
			want: map[string]interface{}{"temperature": 7.97, "pressure": 1011.0, "humidity": 85.0},
		},
		{
			name: "negative temperature",
			json: `{"main":{"temp":-0.96,"pressure":1023,"humidity":100}}`,
			want: map[string]interface{}{"temperature": -0.96, "pressure": 1023.0, "humidity": 100.0},
		},
		{
			name: "zero temperature",
			json: `{"main":{"temp":0,"pressure":1023,"humidity":100}}`,
			want: map[string]interface{}{"temperature": 0.0, "pressure": 1023.0, "humidity": 100.0},
		},
		{
			name: "missing temperature",
			json: `{"main":{"pressure":1023,"humidity":100}}`,
			want: map[string]interface{}{"pressure": 1023.0, "humidity": 100.0},
		},
		{
			name: "no main",
			json: `{}`,
			want: map[string]interface{}{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var w weather
			if err := json.Unmarshal([]byte(tc.json), &w); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if got := weatherFields(&w); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("weatherFields() = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestGetWeatherErrorHidesAPIKey(t *testing.T) {
	// Port 1 on loopback refuses connections, so http.Get fails with a *url.Error.
	old := owmBaseURL
	owmBaseURL = "http://127.0.0.1:1/data/2.5/weather"
	defer func() { owmBaseURL = old }()

	cfg := &Config{}
	cfg.WeatherMap.ApiKey = "SECRET-KEY-123"
	_, err := getWeather(cfg)
	if err == nil {
		t.Fatal("getWeather() returned no error")
	}
	if strings.Contains(err.Error(), "SECRET-KEY-123") {
		t.Errorf("error contains the API key: %v", err)
	}
}

func TestGetWeatherNon200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"cod":401,"message":"Invalid API key"}`))
	}))
	defer srv.Close()
	old := owmBaseURL
	owmBaseURL = srv.URL
	defer func() { owmBaseURL = old }()

	cfg := &Config{}
	cfg.WeatherMap.ApiKey = "SECRET-KEY-123"
	_, err := getWeather(cfg)
	if err == nil || !strings.Contains(err.Error(), "status 401") || !strings.Contains(err.Error(), "Invalid API key") {
		t.Errorf("getWeather() error = %v; want status 401 with the API message", err)
	}
	if err != nil && strings.Contains(err.Error(), "SECRET-KEY-123") {
		t.Errorf("error contains the API key: %v", err)
	}
}
