package main

import (
	"encoding/json"
	"reflect"
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
