package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	influxdb "github.com/influxdata/influxdb1-client/v2"
	"gopkg.in/yaml.v3"
)

var influxClient influxdb.Client

// httpClient has a timeout so that a hung connection cannot stop the ticker
// loop for good (http.DefaultClient has none).
var httpClient = &http.Client{Timeout: 30 * time.Second}

// owmBaseURL is the OpenWeatherMap current-weather endpoint (a variable for tests).
var owmBaseURL = "https://api.openweathermap.org/data/2.5/weather"

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	if len(os.Args) != 2 {
		log.Fatalf("Usage: %s <config>", os.Args[0])
	}
	config, err := parseConfig(os.Args[1])
	if err != nil {
		log.Fatalf("loading config failed: %v", err)
	}

	influxClient, err = influxdb.NewHTTPClient(influxdb.HTTPConfig{
		Addr: config.Influx.Server,
	})
	if err != nil {
		log.Println("Error creating InfluxDB Client: ", err.Error())
		return
	}

	logConfig := *config
	logConfig.WeatherMap.ApiKey = "<redacted>"
	logConfig.Influx.Token = "<redacted>"
	log.Printf("Initialized with config: %+v", logConfig)

	ticker := time.NewTicker(config.Interval)
	for {
		select {
		case <-ticker.C:
			err := updateData(config)
			if err != nil {
				log.Printf("updateData: %v", err)
			}
		case s := <-sigChan:
			log.Printf("Received signal %v, terminating", s)
			return
		}
	}
}

func updateData(config *Config) error {
	weather, err := getWeather(config)
	if err != nil {
		return fmt.Errorf("getData: %v", err)
	}
	err = writeToInflux(weather)
	if err != nil {
		return fmt.Errorf("writeToInflux: %v", err)
	}
	return nil
}

func writeToInflux(weather *weather) error {
	// Create a new point batch
	bp, err := influxdb.NewBatchPoints(influxdb.BatchPointsConfig{
		Database:  "sensors",
		Precision: "s",
	})
	if err != nil {
		return err
	}
	tags := map[string]string{"location": "Outdoor"}
	fields := weatherFields(weather)
	if len(fields) == 0 {
		return fmt.Errorf("no measurements in API response")
	}

	log.Printf("writing to influx: %+v", fields)

	pt, err := influxdb.NewPoint("measurements", tags, fields, time.Now())
	if err != nil {
		return err
	}
	bp.AddPoint(pt)

	// Write the batch
	if err := influxClient.Write(bp); err != nil {
		return err
	}
	return nil
}

// weatherFields returns the InfluxDB fields for w. Only values missing from
// the API response are skipped; zero and negative values (e.g. temperatures
// at or below 0 °C) are valid measurements.
func weatherFields(w *weather) map[string]interface{} {
	fields := map[string]interface{}{}
	if w.Main.Pressure != nil {
		fields["pressure"] = *w.Main.Pressure
	}
	if w.Main.Temperature != nil {
		fields["temperature"] = *w.Main.Temperature
	}
	if w.Main.Humidity != nil {
		fields["humidity"] = *w.Main.Humidity
	}
	return fields
}

func getWeather(config *Config) (*weather, error) {
	uri := fmt.Sprintf("%s?units=metric&lat=%f&lon=%f&appid=%s", owmBaseURL, config.WeatherMap.Latitude, config.WeatherMap.Longitude, config.WeatherMap.ApiKey)
	res, err := httpClient.Get(uri)
	if err != nil {
		// The *url.Error contains the full URL including appid=<key>;
		// return only the underlying error so the key never reaches the log.
		var uerr *url.Error
		if errors.As(err, &uerr) {
			err = uerr.Err
		}
		return nil, fmt.Errorf("GET weather: %v", err)
	}

	data, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, err
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET weather: status %d: %.200s", res.StatusCode, data)
	}

	var w weather
	err = json.Unmarshal(data, &w)
	if err != nil {
		return nil, err
	}

	return &w, nil
}

type weather struct {
	Main struct {
		Temperature *float64 `json:"temp"`
		Humidity    *float64 `json:"humidity"`
		Pressure    *float64 `json:"pressure"`
	} `json:"main"`
}

func kelvinToCelsius(temp float64) float64 {
	return temp - 273.15
}

// parseConfig reads config file at path and returns the content or an error
func parseConfig(path string) (*Config, error) {
	buf, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var config Config
	err = yaml.Unmarshal(buf, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// Config represents a config file
type Config struct {
	Influx struct {
		Server string `yaml:"server"`
		Token  string `yaml:"token"`
	} `yaml:"influx"`
	WeatherMap struct {
		Latitude  float64 `yaml:"latitude"`
		Longitude float64 `yaml:"longitude"`
		ApiKey    string  `yaml:"apikey"`
	} `yaml:"weathermap"`
	Interval time.Duration `yaml:"interval"`
}
