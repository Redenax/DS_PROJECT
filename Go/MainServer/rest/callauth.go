package rest

import (
	"MainServer/config"
	database "MainServer/database_main"
	"MainServer/strutture"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/sony/gobreaker"
)

var breaker *gobreaker.CircuitBreaker

// all'avvio del programma prima che il package venga utilizzato.
func init() {
	// Imposta l'output del logger sulla standard output (stdout).
	log.SetOutput(os.Stdout)

	// Ottieni le impostazioni del circuit breaker chiamando la funzione GetBreakerSettings del pacchetto utility.
	breaker = config.GetBreakerSettings("BreakerConnectionToAuth")
}

// getAuthServer restituisce l'hostname del server di autenticazione letto dalla configurazione.
func getAuthServer() (string, error) {
	// Ottieni l'host e la porta dal file di configurazione.
	host, err := config.GetParametroFromConfig("serverauthhost")
	port, err1 := config.GetParametroFromConfig("serverauthport")

	// Componi l'hostname usando l'host e la porta ottenuti dalla configurazione.
	hostname := fmt.Sprintf("%s:%s", host, port)

	// Gestisci gli errori nel caso in cui la configurazione non contenga i parametri attesi.
	if err != nil {
		log.Println("Errore nel recupero dell'host dal file di configurazione:", err)
		return "", err
	}
	if err1 != nil {
		log.Println("Errore nel recupero della porta dal file di configurazione:", err1)
		return "", err1
	}

	// Restituisci l'hostname ottenuto dalla configurazione.
	return hostname, nil
}

// HandleRegisterRequest è un gestore di richieste HTTP per un processo di registrazione.
func HandleRegisterRequest(w http.ResponseWriter, r *http.Request) {

	// Creare una mappa vuota per i dati JSON e una nuova istanza utente
	datiJSON := make(map[string]interface{})
	utente := strutture.NewUtenti()

	// Ottenere il nome host del server di autenticazione
	hostname, err := getAuthServer()
	if err != nil {
		log.Println("Registrazione - Errore Generico", http.StatusInternalServerError)
		http.Error(w, "Registrazione - Errore Generico", http.StatusInternalServerError)
		return
	}

	// Effettuare una richiesta POST e gestire gli errori
	datiJSON, err1 := PostMethod(w, r, datiJSON, 0)
	if err1 != nil {
		log.Println(err1.Error(), http.StatusInternalServerError)
		http.Error(w, err1.Error(), http.StatusInternalServerError)
		return
	}

	// Inizializzare l'utente con i dati dalla richiesta JSON
	utente.Initialize(
		datiJSON["Nome"].(string),
		datiJSON["Cognome"].(string),
		datiJSON["Email"].(string),
		datiJSON["Password"].(string),
		datiJSON["Id_tg"].(string),
		datiJSON["active"].(string),
	)

	// Convertire i dati dell'utente in formato JSON
	jsonData1, err := manualMarshalJSON(utente)
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Creare una nuova richiesta HTTP POST per l'endpoint dell'API di registrazione
	req, err := http.NewRequest("POST", "http://"+hostname+"/api/register", bytes.NewBuffer(jsonData1))
	if err != nil {
		log.Println(err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Eseguire la richiesta REST con il pattern circuit breaker
	resp, err := EseguireRest(req, breaker)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Chiudere il corpo della risposta quando fatto
	defer resp.Body.Close()

	// Leggere il corpo della risposta
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Errore nella lettura della risposta:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Registrare il codice di stato e il corpo della risposta, quindi scrivere la risposta al client
	log.Println("Codice di Stato:", resp.StatusCode, "\n"+string(body))
	w.WriteHeader(resp.StatusCode)
	w.Write(body)

}

// HandleAuthRequest è un gestore di richieste HTTP per un processo di login.
func HandleAuthRequest(w http.ResponseWriter, r *http.Request) {
	// Creazione di un'istanza di autenticazione e una mappa per i dati JSON.
	auth := strutture.NewAuthentication()
	datiJSON := make(map[string]interface{})

	// Ottenimento dell'hostname del server di autenticazione.
	hostname, err := getAuthServer()
	if err != nil {
		log.Println("Login - Errore Generico", http.StatusInternalServerError)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Esecuzione del metodo POST per ottenere i dati JSON e gestire eventuali errori.
	datiJSON, err1 := PostMethod(w, r, datiJSON, 0)
	if err1 != nil {
		log.Println(err1.Error(), http.StatusInternalServerError)
		http.Error(w, err1.Error(), http.StatusInternalServerError)
		return
	}

	// Inizializzazione dell'istanza di autenticazione con i dati JSON ottenuti.
	auth.Initialize(
		datiJSON["Email"].(string),
		datiJSON["Password"].(string),
		datiJSON["Id_tg"].(string),
	)

	// Assegnazione delle credenziali di autenticazione.
	username := auth.GetEmail()
	password := auth.GetPassword()

	// Creazione della stringa di autorizzazione per l'header.
	res, err1 := json.Marshal(auth.GetIdTg())
	if err1 != nil {
		http.Error(w, err1.Error(), http.StatusBadRequest)
		return
	}
	authString := fmt.Sprintf("%s:%s", username, password)
	base64AuthString := base64.StdEncoding.EncodeToString([]byte(authString))
	authHeader := fmt.Sprintf("Basic %s", base64AuthString)

	// Creazione della richiesta HTTP con le credenziali di login.
	req, err := http.NewRequest("POST", "http://"+hostname+"/api/protetta", bytes.NewBuffer([]byte(res)))
	if err != nil {
		log.Println("Login - Errore nella creazione della richiesta:", err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	req.Header.Add("Authorization", authHeader)

	// Esecuzione della richiesta e gestione di eventuali errori.
	resp, err := EseguireRest(req, breaker)
	if err1 != nil || err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Chiusura del corpo della risposta in modo defer per rilasciare le risorse.
	defer resp.Body.Close()

	// Lettura del corpo della risposta.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Login - Errore nella lettura della risposta:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Scrittura dello status code e del corpo della risposta nella risposta HTTP.
	log.Println("Status Code :", resp.StatusCode, "\n"+(string(body)))
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

// HandleGetTg è un gestore di richieste HTTP per ottenere gli id_tg.
func HandleGetTg(w http.ResponseWriter, r *http.Request) {

	// Ottenimento dell'hostname del server di autenticazione.
	hostname, err := getAuthServer()
	if err != nil {
		log.Println("GetTg - Errore Generico", http.StatusInternalServerError)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Verifica del metodo della richiesta HTTP.
	if r.Method != http.MethodGet {
		http.Error(w, "Metodo non consentito", http.StatusMethodNotAllowed)
		return
	}

	// Ottenimento della lista dagli utenti dal database.
	list, err01 := database.GetEmailRoute()
	if err01 != nil {
		http.Error(w, "errore db sql", http.StatusBadRequest)
		return
	}

	// Conversione della lista in formato JSON.
	res, err1 := json.Marshal(&list)
	if err1 != nil {
		http.Error(w, err1.Error(), http.StatusBadRequest)
		return
	}

	// Creazione di una richiesta HTTP POST per ottenere gli id_tg dal server di autenticazione.
	req, err := http.NewRequest("POST", "http://"+hostname+"/api/getidtg", bytes.NewBuffer([]byte(res)))
	if err != nil {
		log.Println("Login - Errore nella creazione della richiesta:", err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Esecuzione della richiesta e gestione degli errori.
	resp, err := EseguireRest(req, breaker)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Chiusura del corpo della risposta in modo defer per rilasciare le risorse.
	defer resp.Body.Close()

	// Lettura del corpo della risposta.
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Login - Errore nella lettura della risposta:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Println("Status Code :", resp.StatusCode, "\n"+(string(body)))
	// Scrittura dello status code e del corpo della risposta nella risposta HTTP.
	w.WriteHeader(resp.StatusCode)
	w.Write(body)
}

// HandleGetUserData è un gestore di richieste HTTP per ottenere gli id_tg.
func HandleGetUserData(w http.ResponseWriter, r *http.Request) {

	// Ottenimento dell'hostname del server di autenticazione.
	hostname, err := getAuthServer()
	if err != nil {
		log.Println("GetTg - Errore Generico", http.StatusInternalServerError)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	body, err := BodyPost(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}

	// Creazione di una richiesta HTTP POST per ottenere gli id_tg dal server di autenticazione.
	req, err := http.NewRequest("POST", "http://"+hostname+"/api/getuser", bytes.NewBuffer(body))
	if err != nil {
		log.Println("Login - Errore nella creazione della richiesta:", err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Esecuzione della richiesta e gestione degli errori.
	resp, err := EseguireRest(req, breaker)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Chiusura del corpo della risposta in modo defer per rilasciare le risorse.
	defer resp.Body.Close()

	// Lettura del corpo della risposta.
	body1, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Login - Errore nella lettura della risposta:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Println("Status Code :", resp.StatusCode, "\n"+(string(body1)))
	// Scrittura dello status code e del corpo della risposta nella risposta HTTP.
	w.WriteHeader(resp.StatusCode)
	w.Write(body1)
}
func HandleUpdateUserData(w http.ResponseWriter, r *http.Request) {

	// Ottenimento dell'hostname del server di autenticazione.
	hostname, err := getAuthServer()
	if err != nil {
		log.Println("GetTg - Errore Generico", http.StatusInternalServerError)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	body, err := BodyPost(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}

	// Creazione di una richiesta HTTP POST per ottenere gli id_tg dal server di autenticazione.
	req, err := http.NewRequest("POST", "http://"+hostname+"/api/updateuser", bytes.NewBuffer(body))
	if err != nil {
		log.Println("Login - Errore nella creazione della richiesta:", err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Esecuzione della richiesta e gestione degli errori.
	resp, err := EseguireRest(req, breaker)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Chiusura del corpo della risposta in modo defer per rilasciare le risorse.
	defer resp.Body.Close()

	// Lettura del corpo della risposta.
	body1, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Login - Errore nella lettura della risposta:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Println("Status Code :", resp.StatusCode, "\n"+(string(body1)))
	// Scrittura dello status code e del corpo della risposta nella risposta HTTP.
	w.WriteHeader(resp.StatusCode)
	w.Write(body1)
}
func HandleDeleteUserData(w http.ResponseWriter, r *http.Request) {

	datiJSON := make(map[string]interface{})
	auth := strutture.NewAuthentication()
	// Ottenimento dell'hostname del server di autenticazione.
	hostname, err := getAuthServer()
	if err != nil {
		log.Println("GetTg - Errore Generico", http.StatusInternalServerError)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	body, err := BodyPost(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}
	datiJSON, err = MarshalUserPost(body, datiJSON)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	// Inizializzazione dell'istanza di autenticazione con i dati JSON ottenuti.
	auth.Initialize(
		datiJSON["Email"].(string),
		datiJSON["Password"].(string),
		datiJSON["Id_tg"].(string),
	)
	// Creazione di una richiesta HTTP POST per ottenere gli id_tg dal server di autenticazione.
	req, err := http.NewRequest("POST", "http://"+hostname+"/api/deleteuser", bytes.NewBuffer(body))
	if err != nil {
		log.Println("Login - Errore nella creazione della richiesta:", err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// Esecuzione della richiesta e gestione degli errori.
	resp, err := EseguireRest(req, breaker)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	_, err2 := database.DeleteRouteEmail(auth)
	if err2 != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	// Chiusura del corpo della risposta in modo defer per rilasciare le risorse.
	defer resp.Body.Close()

	// Lettura del corpo della risposta.
	body1, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println("Login - Errore nella lettura della risposta:", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Println("Status Code :", resp.StatusCode, "\n"+(string(body1)))
	// Scrittura dello status code e del corpo della risposta nella risposta HTTP.
	w.WriteHeader(resp.StatusCode)
	w.Write(body1)
}
