package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)
var t float64

func main() {
	res, err := http.Get("https://api.openweathermap.org/data/2.5/weather?q=munich&appid=3bf536d618ecc7d7eac5b7586d1edcea")
	if err != nil {
		log.Fatal(err)
	}

	data, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		log.Fatal(err)
	}

	var w weather
	err = json.Unmarshal(data, &w)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("temperature = %.1f\n", kelvinToCelsius(w.Main.Temp))
}

type weather struct {
	Main struct {
		Temp float64 `json:"temp"`
	} `json:"main"`
}

func kelvinToCelsius(t float64) float64 {
	return t - 273.15
}

// t = temp - 273.15

// t = temp - 273.15
