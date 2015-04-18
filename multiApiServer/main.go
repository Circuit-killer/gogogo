package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

type weatherProvider interface {
	temperature(city string) (float64, error)
}
type multiWeatherProvider []weatherProvider

type openWeatherMap struct{}

func (w openWeatherMap) temperature(city string) (float64, error) {
	resp, err := http.Get("http://api.openweathermap.org/data/2.5/weather?q=" + city)

	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	var d struct {
		Main struct {
			Kelvin float64 `json:"temp"`
		} `json:"Main"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return 0, err
	}
	Celsius := d.Main.Kelvin - 273.15
	log.Printf("openWeatherMap: %s: %.2f", city, Celsius)
	return Celsius, nil
}


type weatherUnderground struct{
	apiKey string
}

func (w weatherUnderground) temperature(city string) (float64, error) {
	resp, err := http.Get("http://api.wunderground.com/api/" + w.apiKey + "/conditions/q/" + city + ".json")

	if err != nil {
		// log.Printf("Error");
		return 0, err
	}

	defer resp.Body.Close()

	var d struct {
		Observation struct {
			Celsius float64 `json:"temp_c"`
		} `json:"current_observation"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		// log.Printf("ERROR");
		return 0, err
	}

	c := d.Observation.Celsius
	log.Printf("weatherUnderground: %s: %.2f", city, c)
	return c, nil
}

func (w multiWeatherProvider) temperature(city string) (float64, error) {
	sum := 0.0;
	for _, provider := range w {
		k, err := provider.temperature(city)
		if err != nil {
			return 0, err
		}

		sum += k
	}

	return sum / float64(len(w)), nil
}


func main() {
	mw := multiWeatherProvider{
		openWeatherMap{},
		weatherUnderground{apiKey: "3b67e0c878ad6999"},
	}

	http.HandleFunc("/weather/", func(w http.ResponseWriter, r *http.Request) {
		begin := time.Now()
		city := strings.SplitN(r.URL.Path, "/", 3)[2]

		temp, err := mw.temperature(city)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"city": city,
			"temp": temp,
			"took": time.Since(begin).String(),
			})
	})

	http.ListenAndServe(":8080", nil)
}