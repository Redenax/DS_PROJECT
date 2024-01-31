package rest

import (
	database "MainServer/database_main"
	"MainServer/strutture"
	"encoding/json"
	"log"
	"net/http"
)

// HandleRegisterRouteRequest è un gestore di richieste HTTP per la registrazione di una route.
func HandleRegisterRouteRequest(w http.ResponseWriter, r *http.Request) {
	// Creazione di una mappa per i dati JSON e di un'istanza di Routes.
	datiJSON := make(map[string]interface{})
	route := strutture.NewRoutes()

	// Esecuzione del metodo POST per ottenere i dati JSON e gestire eventuali errori.
	datiJSON, err := PostMethod(w, r, datiJSON, 1)
	if err != nil {
		log.Println("Registrazione - Errore Generico", http.StatusInternalServerError)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Inizializzazione dell'istanza di Routes con i dati JSON ottenuti.
	route.Initialize(
		datiJSON["Nome"].(string),
		datiJSON["Partenza"].(string),
		datiJSON["Destinazione"].(string),
		datiJSON["Email"].(string),
	)

	// Creazione del percorso nel database e gestione di eventuali errori.
	resp2, err1 := database.CreateRoute(route)
	if err1 != nil {
		http.Error(w, "errore db sql", http.StatusInternalServerError)
		return
	}
	log.Println((string(*resp2)))
	// Scrittura della risposta come stringa nella risposta HTTP.
	w.Write([]byte(*resp2))
}

// HandleDeleteRouteRequest è un gestore di richieste HTTP per la cancellazione di una route.
func HandleDeleteRouteRequest(w http.ResponseWriter, r *http.Request) {
	// Creazione di una mappa per i dati JSON e di un'istanza di Routes.
	datiJSON := make(map[string]interface{})
	route := strutture.NewRoutes()

	// Esecuzione del metodo POST per ottenere i dati JSON e gestire eventuali errori.
	datiJSON, err1 := PostMethod(w, r, datiJSON, 1)
	if err1 != nil {
		log.Println("Cancellazione - Errore Generico", http.StatusInternalServerError)
		http.Error(w, err1.Error(), http.StatusInternalServerError)
		return
	}

	// Inizializzazione dell'istanza di Routes con i dati JSON ottenuti.
	route.Initialize(
		datiJSON["Nome"].(string),
		datiJSON["Partenza"].(string),
		datiJSON["Destinazione"].(string),
		datiJSON["Email"].(string),
	)

	// Cancellazione del percorso nel database e gestione di eventuali errori.
	resp2, err1 := database.DeleteRoute(route)
	if err1 != nil {
		http.Error(w, "errore db sql", http.StatusInternalServerError)
		return
	}
	log.Println((string(*resp2)))
	// Scrittura della risposta come stringa nella risposta HTTP.
	w.Write([]byte(*resp2))
}

// HandleEnableRouteRequest è un gestore di richieste HTTP per abilitare una route.
func HandleEnableRouteRequest(w http.ResponseWriter, r *http.Request) {
	// Creazione di una mappa per i dati JSON e di un'istanza di Routes.
	datiJSON := make(map[string]interface{})
	route := strutture.NewRoutes()

	// Esecuzione del metodo POST per ottenere i dati JSON e gestire eventuali errori.
	datiJSON, err1 := PostMethod(w, r, datiJSON, 1)
	if err1 != nil {
		log.Println("Abilitazione - Errore Generico", http.StatusInternalServerError)
		http.Error(w, err1.Error(), http.StatusInternalServerError)
		return
	}

	// Inizializzazione dell'istanza di Routes con i dati JSON ottenuti.
	route.Initialize(
		datiJSON["Nome"].(string),
		datiJSON["Partenza"].(string),
		datiJSON["Destinazione"].(string),
		datiJSON["Email"].(string),
	)

	// Abilitazione dei percorsi nel database e gestione di eventuali errori.
	resp2, err1 := database.EnableRoutes(route)
	if err1 != nil {
		http.Error(w, "errore db sql", http.StatusInternalServerError)
		return
	}

	// Verifica se la lista di percorsi abilitati è vuota.
	if len(resp2) == 0 {
		http.Error(w, "Lista vuota", http.StatusAccepted)
		return
	}

	// Conversione della lista di percorsi abilitati in formato JSON.
	jsonData, err := json.Marshal(resp2)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Println((string(jsonData)))
	// Scrittura della risposta come stringa nella risposta HTTP.
	w.Write(jsonData)
}

// HandleDisableRouteRequest è un gestore di richieste HTTP per disabilitare una ruote.
func HandleDisableRouteRequest(w http.ResponseWriter, r *http.Request) {
	// Creazione di una mappa per i dati JSON e di un'istanza di Routes.
	datiJSON := make(map[string]interface{})
	route := strutture.NewRoutes()

	// Esecuzione del metodo POST per ottenere i dati JSON e gestire eventuali errori.
	datiJSON, err1 := PostMethod(w, r, datiJSON, 1)
	if err1 != nil {
		log.Println(err1)
		http.Error(w, err1.Error(), http.StatusInternalServerError)
		return
	}

	// Inizializzazione dell'istanza di Routes con i dati JSON ottenuti.
	route.Initialize(
		datiJSON["Nome"].(string),
		datiJSON["Partenza"].(string),
		datiJSON["Destinazione"].(string),
		datiJSON["Email"].(string),
	)

	// Disabilitazione dei percorsi nel database e gestione di eventuali errori.
	resp2, err1 := database.DisableRoutes(route)
	if err1 != nil {
		http.Error(w, "errore db sql", http.StatusInternalServerError)
		return
	}
	log.Println((string(resp2)))
	// Scrittura della risposta come stringa nella risposta HTTP.
	w.Write([]byte(resp2))
}

// HandleGetProvince è un gestore di richieste HTTP per ottenere le province.
func HandleGetProvince(w http.ResponseWriter, r *http.Request) {
	// Verifica del metodo della richiesta HTTP.
	if r.Method != http.MethodGet {
		http.Error(w, "Get Province - Metodo non consentito", http.StatusMethodNotAllowed)
		return
	}

	// Ottenimento delle province dal database e gestione di eventuali errori.
	resp2, err1 := database.GetProvince()
	if err1 != nil {
		http.Error(w, "errore db sql", http.StatusInternalServerError)
		return
	}

	// Verifica se la lista di province è vuota.
	if len(resp2) == 0 {
		http.Error(w, "Lista vuota", http.StatusBadRequest)
		return
	}

	// Conversione della lista di province in formato JSON.
	jsonData, err := json.Marshal(resp2)
	if err != nil {
		http.Error(w, "Get Province  - Errore nel marshalling dei dati in JSON", http.StatusInternalServerError)
		return
	}
	log.Println((string(jsonData)))
	// Scrittura della risposta come stringa nella risposta HTTP.
	w.Write([]byte(jsonData))
}

// HandleGetRoute è un gestore di richieste HTTP per ottenere le route di un utente.
func HandleGetRoute(w http.ResponseWriter, r *http.Request) {
	// Creazione di una mappa per i dati JSON e una variabile per l'email.
	datiJSON := make(map[string]interface{})
	var email string

	// Esecuzione del metodo POST per ottenere i dati JSON e gestire eventuali errori.
	datiJSON, err1 := PostMethod(w, r, datiJSON, 1)
	if err1 != nil {
		log.Println("Lettura Route - Errore Generico", http.StatusInternalServerError)
		http.Error(w, "Lettura Route - Errore Generico", http.StatusInternalServerError)
		return
	}

	// Estrazione dell'email dai dati JSON.
	email = datiJSON["Email"].(string)

	// Ottenimento delle route dall'email nel database e gestione di eventuali errori.
	resp2, err1 := database.GetRoute(email)
	if err1 != nil {
		http.Error(w, "errore db sql", http.StatusInternalServerError)
		return
	}

	// Verifica se la lista di route è vuota.
	if len(resp2) == 0 {
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("Lista vuota"))
		return
	}

	// Conversione della lista di route in formato JSON.
	jsonData, err := json.Marshal(resp2)
	if err != nil {
		http.Error(w, "Lettura Route - Errore nel marshalling dei dati in JSON", http.StatusInternalServerError)
		return
	}
	log.Println((string(jsonData)))
	// Scrittura della risposta come stringa nella risposta HTTP.
	w.Write([]byte(jsonData))
}
