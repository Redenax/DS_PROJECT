package main

import (
	database "Server_Auth/database_auth"
	strutture "Server_Auth/strutture"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Funzione di marshaling manuale
func manualMarshalJSON(u strutture.Registrazione) ([]byte, error) {
	jsonData := fmt.Sprintf(`{"Nome":"%s","Cognome":"%s","Email":"%s","Password":"%s","Id_tg":"%s","Active":%s}`,
		u.GetNome(), u.GetCognome(), u.GetEmail(), u.GetPassword(), u.GetIdTg(), u.GetActive())

	return []byte(jsonData), nil
}

// PostMethod gestisce le richieste HTTP di tipo POST.
func PostMethod(w http.ResponseWriter, r *http.Request, datiJSON map[string]interface{}) (map[string]interface{}, error) {
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

	// Chiudi il corpo della richiesta in modo defer per rilasciare le risorse.
	defer r.Body.Close()

	datiJSON, err = MarshalUserPost(body, datiJSON)

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

// Configura le credenziali di autenticazione di base
func middlewareAutenticazioneBase(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		utenti := strutture.NewUtenti()

		// Ottiene le credenziali di autenticazione dall'intestazione Authorization
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			unauthorized(w)
			return
		}

		// Verifica le credenziali di autenticazione di base
		auth := strings.SplitN(authHeader, " ", 2)
		if len(auth) != 2 || auth[0] != "Basic" {
			unauthorized(w)
			return
		}

		payload, err := base64.StdEncoding.DecodeString(auth[1])
		if err != nil {
			fmt.Println("errore decode")
			unauthorized(w)
			return
		}

		pair := strings.SplitN(string(payload), ":", 2)

		utenti.SetEmail(pair[0])

		var temp string

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Errore nella lettura del corpo della richiesta", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		// Deserializza il corpo JSON in un oggetto Person

		err = json.Unmarshal(body, &temp)
		if err != nil {
			http.Error(w, "Errore nella deserializzazione JSON", http.StatusBadRequest)
			return
		}
		fmt.Println("pEmail: " + pair[0] + " pPassword:" + pair[1] + " pID:" + temp)
		utenti.SetPassword(pair[1])
		utenti.SetIdTg(temp)
		fmt.Println("Email: " + utenti.GetEmail() + " Password:" + utenti.GetPassword() + " ID:" + utenti.GetIdTg())
		res, err := database.ReadeUser(utenti)
		if res == nil {
			unauthorized(w)
			return
		}
		if err != nil {
			http.Error(w, "Errore Lettura ", http.StatusBadRequest)
			return
		}
		if len(pair) != 2 || pair[0] != res.GetEmail() || pair[1] != res.GetPassword() {

			unauthorized(w)
			return
		}

		// Chiama la prossima funzione nella catena di middleware
		next.ServeHTTP(w, r)
	})
}

func middlewareRegister(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		datiJSON := make(map[string]interface{})
		utenti := strutture.NewUtenti()
		var err1 error

		datiJSON, err1 = PostMethod(w, r, datiJSON)

		if err1 != nil {
			log.Println(err1.Error(), http.StatusInternalServerError)
			http.Error(w, err1.Error(), http.StatusInternalServerError)
			return
		}

		utenti.Initialize(
			datiJSON["Nome"].(string),
			datiJSON["Cognome"].(string),
			datiJSON["Email"].(string),
			datiJSON["Password"].(string),
			datiJSON["Id_tg"].(string),
			datiJSON["active"].(string),
		)
		fmt.Println(utenti.GetNome())

		res, err := database.CreateUser(utenti)
		if res == nil {
			erroreRegister(w)
		}
		if err != nil {
			http.Error(w, "Errore Creazione ", http.StatusBadRequest)
			return
		}

		// Chiama la prossima funzione nella catena di middleware
		next.ServeHTTP(w, r)
	})
}

func middlewareTg(w http.ResponseWriter, r *http.Request) {

	var a []string
	if r.Method != http.MethodPost {
		http.Error(w, "Metodo non consentito", http.StatusMethodNotAllowed)
		return
	}

	// Leggi il corpo della richiesta
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Errore nella lettura del corpo della richiesta", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Deserializza il corpo JSON in un oggetto
	fmt.Println(string(body))
	err = json.Unmarshal(body, &a)
	if err != nil {
		http.Error(w, "Errore nella deserializzazione JSON", http.StatusBadRequest)
		return
	}
	fmt.Println(a)
	//----------------

	res, err := database.GetEmailRoute(a)
	if len(res) == 0 || res[0] == "" {
		http.Error(w, "Lista vuota", http.StatusOK)
		return
	}
	if err != nil {
		http.Error(w, "Errore Creazione ", http.StatusBadRequest)
		return
	}
	jsonData, err10 := json.Marshal(res)
	if err10 != nil {
		http.Error(w, "Attivazione Route - Errore nel marshalling dei dati in JSON", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func middlewareGetUser(w http.ResponseWriter, r *http.Request) {
	datiJSON := make(map[string]interface{})
	utente := strutture.NewUtenti()

	datiJSON, err := PostMethod(w, r, datiJSON)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//----------------
	utente.Initialize(
		datiJSON["Nome"].(string),
		datiJSON["Cognome"].(string),
		datiJSON["Email"].(string),
		datiJSON["Password"].(string),
		datiJSON["Id_tg"].(string),
		datiJSON["active"].(string),
	)
	res, err := database.ReadeUser(utente)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonData, err10 := manualMarshalJSON(res)
	if err10 != nil {
		http.Error(w, "Errore nel marshalling dei dati in JSON", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func convertMapToJSON(data map[string]interface{}) []byte {
	result, _ := json.Marshal(data)
	return result
}

func middlewareUpdatetUser(w http.ResponseWriter, r *http.Request) {

	utente := strutture.NewUtenti()
	oldutente := strutture.NewUtenti()
	var utentiList []map[string]interface{}

	if r.Method != http.MethodPost {
		log.Println("Metodo non consentito", http.StatusMethodNotAllowed)
		http.Error(w, "metodo non consentito", http.StatusInternalServerError)
		return

	}

	// Leggi il corpo della richiesta.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("errore nella lettura del corpo della richiesta", http.StatusBadRequest)
		err = errors.New("errore nella lettura del corpo della richiesta")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Chiudi il corpo della richiesta in modo defer per rilasciare le risorse.
	defer r.Body.Close()

	// Deserializza la stringa JSON nella lista di oggetti Utente

	err = json.Unmarshal(body, &utentiList)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for i, utenteData := range utentiList {
		if i == 0 {

			oldutente.UnmarshalJSON(convertMapToJSON(utenteData))
		}
		if i == 1 {
			utente.UnmarshalJSON(convertMapToJSON(utenteData))
		}

	}
	utente.ToString()
	oldutente.ToString()

	res, err := database.UpdateUser(utente, oldutente)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonData, err10 := manualMarshalJSON(res)
	if err10 != nil {
		http.Error(w, "Errore nel marshalling dei dati in JSON", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func middlewareDeleteUser(w http.ResponseWriter, r *http.Request) {
	datiJSON := make(map[string]interface{})
	utente := strutture.NewUtenti()

	datiJSON, err := PostMethod(w, r, datiJSON)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//----------------
	utente.Initialize(
		datiJSON["Nome"].(string),
		datiJSON["Cognome"].(string),
		datiJSON["Email"].(string),
		datiJSON["Password"].(string),
		datiJSON["Id_tg"].(string),
		datiJSON["active"].(string),
	)
	res, err := database.DeleteUser(utente)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(*res))
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("401 Unauthorized"))
}

func gestoreProtetto(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Authorized"))
}

func erroreRegister(w http.ResponseWriter) {

	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("400 Bad Request - Errore Registrazione\n"))
}

func okRegister(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Registrazione riuscita!"))
}

func main() {
	fmt.Println("Start Server Autenticazione")

	fmt.Println("Avvio Database connection tra 1 minuto...")
	time.Sleep(60 * time.Second)
	fmt.Println("Avviato")
	database.StartDBUtenti()

	// Configura il router e aggiungi il middleware di autenticazione di base
	go func() {

		mux := http.NewServeMux()

		// Gestore per l'API protetta con middleware di autenticazione di base
		mux.Handle("/api/protetta", middlewareAutenticazioneBase(http.HandlerFunc(gestoreProtetto)))
		// Gestore per l'API di registrazione con middleware specifico
		mux.Handle("/api/register", middlewareRegister(http.HandlerFunc(okRegister)))
		mux.HandleFunc("/api/getidtg", middlewareTg)
		mux.HandleFunc("/api/getuser", middlewareGetUser)
		mux.HandleFunc("/api/updateuser", middlewareUpdatetUser)
		mux.HandleFunc("/api/deleteuser", middlewareDeleteUser)
		// Avvia il server sulla porta 8081 con il mux come router
		fmt.Println("Server in ascolto su http://localhost:8081")
		http.ListenAndServe(":8081", mux)
	}()
	select {}
}
