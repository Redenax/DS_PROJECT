package main

import (
	database "MainServer/database_main"
	"MainServer/kafka"
	"MainServer/rest"
	"MainServer/routes"
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type Data struct {
	NomeTratta string `json:"NomeTratta"`
	Messaggio  string `json:"Messaggio"`
}

// all'avvio del programma prima che il package venga utilizzato.
func init() {
	// Imposta l'output del logger sulla standard output (stdout).
	log.SetOutput(os.Stdout)
}

func readProvincesFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var provinces []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		province := strings.TrimSpace(scanner.Text())
		if province != "" {
			provinces = append(provinces, province)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return provinces, nil
}

// CallMapsApi è una funzione che esegue chiamate all'API di mappe per ogni coppia di province, ottiene la tratta e inviare messaggi al sistema Kafka.
func CallMapsApi(province []string) {
	dati := make(map[string]string)
	topic := "tratte-auto"
	for i := range province {

		for j := range province {
			// Verifica se le province sono diverse.
			if province[i] != province[j] {
				// Conversione delle province in stringhe.
				a := string(province[i])
				b := string(province[j])

				// Chiamata all'API di mappe per ottenere la tratta tra le province.
				resp := routes.Routing(a, b)
				if resp == nil {
					log.Println("Errore caricamento route")
					return
				}

				// Iterazione sulla rotta ottenuta dalla risposta.
				for _, s := range resp.Routes {
					// Nome del topic basato sulle province.
					nome_tratta := fmt.Sprintf("%s_%s", a, b)

					// Creazione del messaggio da inviare al sistema Kafka nel topic specifico.
					messaggio := fmt.Sprintf("Per il percoso attuale: #%s ,la durata Attuale è: %d minuti, La durata Tipica è: %d minuti", nome_tratta, ((s.Duration.Seconds) / 60), ((s.StaticDuration.Seconds) / 60))
					dati[nome_tratta] = messaggio
				}
			}
		}
	}
	// Marshal the map to JSON
	marshaledData, err := json.Marshal(dati)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	// Invio del messaggio al sistema Kafka.
	kafka.KafkaProducer(topic, marshaledData)

}

func main() {
	// Definizione delle province e dell'intervallo di tempo.
	filePath := "province.txt"
	province, err := readProvincesFromFile(filePath)
	if err != nil {
		fmt.Println("Errore nella lettura del file:", err)
		return
	}

	fmt.Println("Province lette dal file:", province)

	intervallo := 1 * time.Hour

	// Goroutine per avviare Kafka con un ritardo di 30 secondi.
	go func() {
		log.Println("Avvio Kafka Startup tra 30 secondi...")
		time.Sleep(30 * time.Second)
		log.Println("Avviato")
		CallMapsApi(province)
		for {
			time.Sleep(intervallo)
			CallMapsApi(province)
		}
	}()

	// Attendi 1 minuto prima di avviare il database.
	log.Println("Avvio DB tra 1 minuto...")
	time.Sleep(1 * time.Minute)
	log.Println("Avviato")
	database.StartDBRoute(province)

	// Configurazione del server HTTP e del router.
	log.Println("Avvio Server in ascolto")
	mux := http.NewServeMux()

	// Configurazione del router per l'autenticazione
	mux.HandleFunc("/api/v1/authentication", rest.HandleAuthRequest)
	mux.HandleFunc("/api/v1/register", rest.HandleRegisterRequest)
	mux.HandleFunc("/api/v1/gettg", rest.HandleGetTg)
	mux.HandleFunc("/api/v1/getuserdata", rest.HandleGetUserData)
	mux.HandleFunc("/api/v1/updatedata", rest.HandleUpdateUserData)
	mux.HandleFunc("/api/v1/deleteuser", rest.HandleDeleteUserData)
	// Configurazione del router per le tratte
	mux.HandleFunc("/api/v1/deletesRoute", rest.HandleDeleteRouteRequest)
	mux.HandleFunc("/api/v1/registerRoute", rest.HandleRegisterRouteRequest)
	mux.HandleFunc("/api/v1/enableRoute", rest.HandleEnableRouteRequest)
	mux.HandleFunc("/api/v1/disableRoute", rest.HandleDisableRouteRequest)
	mux.HandleFunc("/api/v1/getroute", rest.HandleGetRoute)

	// Configurazione del router per le province
	mux.HandleFunc("/api/v1/getprovince", rest.HandleGetProvince)

	// Configurazione e avvio del server HTTP.
	server := &http.Server{
		Addr:    ":25536",
		Handler: mux,
	}

	log.Println("Server in ascolto su http://localhost:25536")
	log.Fatal(server.ListenAndServe())

}
