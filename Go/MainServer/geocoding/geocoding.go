package geocoding

import (
	config "MainServer/config"
	"MainServer/rest"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/sony/gobreaker"
)

var breaker *gobreaker.CircuitBreaker

func init() {
	log.SetOutput(os.Stdout)
	// Inizializza il circuit breaker
	breaker = config.GetBreakerSettings("BreakerConnectionGeocoding")
}

func getCoordinates(breaker *gobreaker.CircuitBreaker, address string, key string) (float64, float64, error) {

	var resp *http.Response
	var response struct {
		Results []struct {
			Geometry struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
		} `json:"results"`
	}
	// Crea l'URL
	url := fmt.Sprintf(
		"https://maps.googleapis.com/maps/api/geocode/json?address=%s&key=%s",
		address,
		key,
	)

	// Crea una richiesta HTTP GET
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Println("Errore Creazione NewRequest")
		return 0, 0, err
	}
	resp, err = rest.EseguireRest(req, breaker)
	if err != nil {
		log.Println("Errore esecuzione risposta HTTP")
		return 0, 0, err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Errore Lettura del corpo della risposta HTTP")
		return 0, 0, err
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Println("Errore Unmarshal del corpo della risposta JSON")
		return 0, 0, err
	}

	// Restituisce le coordinate
	return response.Results[0].Geometry.Location.Lat, response.Results[0].Geometry.Location.Lng, nil
}

func StartUpConnection(Location string) (float64, float64) {
	// Imposta la chiave API di Google Maps
	key, err := config.GetParametroFromConfig("apigoogletoken")
	if err != nil {
		log.Println(err)
	}
	// Ottieni le coordinate
	param := url.PathEscape(Location)
	latitude, longitude, err := getCoordinates(breaker, param, key)
	if err != nil {
		log.Println(err)
	}

	for breaker.State().String() == gobreaker.StateClosed.String() && err != nil {
		latitude, longitude, err = getCoordinates(breaker, param, key)
	}
	if breaker.State().String() != gobreaker.StateClosed.String() {
		// Sleep o aggiungi un ritardo prima della prossima iterazione
		fmt.Println("Stato Aperto " + breaker.Name())
		log.Printf("Errore %s ", err)
		return 0, 0
	}
	// Stampa le coordinate
	fmt.Println(latitude, longitude)

	return latitude, longitude
}
