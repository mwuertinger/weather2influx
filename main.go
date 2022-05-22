package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	influxdb "github.com/influxdata/influxdb1-client/v2"
	"gopkg.in/yaml.v3"
)

var influxClient influxdb.Client

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

	log.Printf("Initialized with config: %v", config)

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
	fields := map[string]interface{}{}
	if weather.Main.Pressure > 0 {
		fields["pressure"] = weather.Main.Pressure
	}
	if weather.Main.Temperature > 0 {
		fields["temperature"] = weather.Main.Temperature
	}
	if weather.Main.Humidity > 0 {
		fields["humidity"] = weather.Main.Humidity
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

func getWeather(config *Config) (*weather, error) {
	uri := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?units=metric&lat=%f&lon=%f&appid=%s", config.WeatherMap.Latitude, config.WeatherMap.Longitude, config.WeatherMap.ApiKey)
	res, err := http.Get(uri)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, err
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
		Temperature float64 `json:"temp"`
		Humidity    float64 `json:"humidity"`
		Pressure    float64 `json:"pressure"`
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
