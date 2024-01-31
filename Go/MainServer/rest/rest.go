package rest

import (
	"MainServer/strutture"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/sony/gobreaker"
)

// eseguireRest è una funzione che esegue una richiesta HTTP con la gestione del circuit breaker.
func EseguireRest(req *http.Request, breaker *gobreaker.CircuitBreaker) (*http.Response, error) {
	var resp *http.Response
	var err error

	// Esegui la richiesta all'interno del circuit breaker.
	_, execErr := breaker.Execute(func() (interface{}, error) {
		client := &http.Client{}
		resp, err = client.Do(req)
		if err != nil {
			log.Println("Errore nell'esecuzione della richiesta:", err)
			return nil, err
		}

		return nil, err
	})

	// Se si verifica un errore nell'esecuzione e lo stato del circuit breaker è aperto, restituisci l'errore.
	if execErr != nil && breaker.State().String() == gobreaker.StateOpen.String() {
		return nil, execErr
	}

	// Loop che continua a eseguire la richiesta fintanto che lo stato del circuit breaker è chiuso e si verifica un errore.
	for breaker.State().String() == gobreaker.StateClosed.String() && execErr != nil {
		_, execErr = breaker.Execute(func() (interface{}, error) {
			client := &http.Client{}
			resp, err = client.Do(req)
			if err != nil {
				log.Println("Errore nell'esecuzione della richiesta:", err)
				return nil, err
			}

			return nil, err
		})
	}

	// Se lo stato del circuit breaker non è chiuso, logga lo stato aperto e restituisci l'errore.
	if breaker.State().String() != gobreaker.StateClosed.String() {
		log.Println("Stato Aperto " + breaker.Name())
		return nil, execErr
	}

	// Restituisci la risposta HTTP.
	return resp, nil
}
func BodyPost(w http.ResponseWriter, r *http.Request) ([]byte, error) {
	// Verifica se il metodo della richiesta è POST.
	if r.Method != http.MethodPost {
		log.Println("Metodo non consentito", http.StatusMethodNotAllowed)
		err := errors.New("metodo non consentito")
		return nil, err
	}

	// Leggi il corpo della richiesta.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("Errore nella lettura del corpo della richiesta", http.StatusBadRequest)
		err := errors.New("errore nella lettura del corpo della richiesta")
		return nil, err
	}
	return body, nil
}

// PostMethod gestisce le richieste HTTP di tipo POST.
func PostMethod(w http.ResponseWriter, r *http.Request, datiJSON map[string]interface{}, value int) (map[string]interface{}, error) {

	body, err := BodyPost(w, r)
	if err != nil {
		log.Println("Errore nella lettura del corpo della richiesta", http.StatusBadRequest)
		err := errors.New("errore nella lettura del corpo della richiesta")
		return nil, err
	}
	// Chiudi il corpo della richiesta in modo defer per rilasciare le risorse.
	if body != nil {
		defer r.Body.Close()
	}

	// Determina la logica di deserializzazione in base al valore di 'value'.
	switch value {
	case 0:
		datiJSON, err = MarshalUserPost(body, datiJSON)
	case 1:
		datiJSON, err = MarshalRoutePost(body, datiJSON)
	default:
		err := errors.New("selezione sbagliata")
		return nil, err
	}

	// Gestisci eventuali errori nella deserializzazione JSON.
	if err != nil {
		log.Println("Errore nella deserializzazione JSON", http.StatusBadRequest)
		err := errors.New("errore nella deserializzazione JSON")
		return nil, err
	}

	// Restituisci i dati JSON deserializzati.
	return datiJSON, nil
}

// MarshalUserPost deserializza il corpo della richiesta JSON per il tipo utente.
func MarshalUserPost(body []byte, datiJSON map[string]interface{}) (map[string]interface{}, error) {
	// Deserializza il corpo della richiesta JSON nel parametro datiJSON.
	err := json.Unmarshal(body, &datiJSON)
	if err != nil {
		log.Println("Registrazione - Errore nella deserializzazione JSON", http.StatusBadRequest)
		return nil, err
	}

	// Verifica e imposta valori predefiniti per i campi opzionali mancanti.
	if datiJSON["Nome"] == nil {
		datiJSON["Nome"] = ""
	}
	if datiJSON["Cognome"] == nil {
		datiJSON["Cognome"] = ""
	}
	if datiJSON["Password"] == nil {
		datiJSON["Password"] = ""
	}
	if datiJSON["Email"] == nil {
		err := errors.New("errore nella deserializzazione json - email")
		return nil, err
	}
	if datiJSON["Id_tg"] == nil {
		datiJSON["Id_tg"] = "nullo"
	}
	if datiJSON["active"] == nil {
		datiJSON["active"] = "0"
	}

	// Restituisci la mappa aggiornata.
	return datiJSON, nil
}

// MarshalRoutePost deserializza il corpo della richiesta JSON per il tipo route.
func MarshalRoutePost(body []byte, datiJSON map[string]interface{}) (map[string]interface{}, error) {
	// Deserializza il corpo della richiesta JSON nel parametro datiJSON.
	err := json.Unmarshal(body, &datiJSON)
	if err != nil {
		log.Println("Errore nella deserializzazione JSON", http.StatusBadRequest)
		return nil, err
	}

	// Verifica e imposta valori predefiniti per i campi opzionali mancanti.
	if datiJSON["Nome"] == nil {
		datiJSON["Nome"] = ""
	}
	if datiJSON["Partenza"] == nil {
		datiJSON["Partenza"] = ""
	}
	if datiJSON["Destinazione"] == nil {
		datiJSON["Destinazione"] = ""
	}
	if datiJSON["Email"] == nil {
		err := errors.New("errore nella deserializzazione json - email")
		return nil, err
	}

	// Restituisci la mappa aggiornata.
	return datiJSON, nil
}

// Funzione di marshaling manuale
func manualMarshalJSON(u strutture.Registrazione) ([]byte, error) {
	jsonData := fmt.Sprintf(`{"Nome":"%s","Cognome":"%s","Email":"%s","Password":"%s","Id_tg":"%s","Active":%s}`,
		u.GetNome(), u.GetCognome(), u.GetEmail(), u.GetPassword(), u.GetIdTg(), u.GetActive())

	return []byte(jsonData), nil
}
