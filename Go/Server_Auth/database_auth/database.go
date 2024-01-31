package database

import (
	config "Server_Auth/config"
	"Server_Auth/strutture"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/sony/gobreaker"
)

const (
	dbname = "utenti"
)

var breaker *gobreaker.CircuitBreaker

func init() {
	log.SetOutput(os.Stdout)
	// Inizializza il circuit breaker
	breaker = config.GetBreakerSettings("BreakerConnectionToSqlAuth")

}

// Check if the error is a MySQL duplicate key violation error.
func isDuplicateKeyError(err error) bool {
	mysqlError, ok := err.(*mysql.MySQLError)
	if ok && mysqlError.Number == 1062 {
		// MySQL error code for duplicate key violation
		return true
	}
	return false
}

func isEmptyError(err error) bool {

	return err == sql.ErrNoRows

}

// connectDB apre una connessione al database utilizzando un circuit breaker per la gestione degli errori.
func connectDB(name string) (*sql.DB, error) {
	// Apre una connessione al database utilizzando un circuit breaker
	db, err := openDBWithBreaker(dsn(name))

	// Riprova a connettersi mentre il circuit breaker è chiuso e si verifica un errore
	for breaker.State().String() == gobreaker.StateClosed.String() && err != nil {
		// Riprova a connettersi
		db, err = openDBWithBreaker(dsn(name))
	}

	// Se il circuit breaker è aperto, restituisci un errore
	if breaker.State().String() != gobreaker.StateClosed.String() {
		log.Println("Stato Aperto " + breaker.Name())
		log.Printf("Error %s ", err)
		return nil, err
	}

	return db, nil
}

// queryDB esegue una query sul database utilizzando un circuit breaker per la gestione degli errori.
func queryDB(db *sql.DB, query string) error {
	// Esegue la query utilizzando un circuit breaker
	err := executeQueryWithBreaker(db, query)

	// Riprova a eseguire la query mentre il circuit breaker è chiuso e si verifica un errore
	for breaker.State().String() == gobreaker.StateClosed.String() && err != nil {
		// Riprova a eseguire la query
		err = executeQueryWithBreaker(db, query)
	}

	// Se il circuit breaker è aperto, restituisci un errore
	if breaker.State().String() != gobreaker.StateClosed.String() {
		log.Println("Stato Aperto " + breaker.Name())
		log.Printf("Error %s ", err)
		return err
	}

	return nil
}

// executeQueryWithBreaker esegue una query sul database utilizzando un circuit breaker per la gestione degli errori.
func executeQueryWithBreaker(db *sql.DB, query string) error {
	// Esegue la query utilizzando un circuit breaker
	_, execErr := breaker.Execute(func() (interface{}, error) {
		// Esegui la query con il contesto di background
		res, err := db.ExecContext(context.Background(), query)

		// Verifica la violazione della chiave duplicata.
		if err != nil {
			return nil, err
		}

		// Ottieni il numero di righe interessate
		rowsAffected, _ := res.RowsAffected()
		log.Printf("Rows affected: %d", rowsAffected)

		return nil, nil
	})

	// Se il circuit breaker è aperto, execErr sarà gobreaker.ErrOpenState
	if execErr != nil {
		return execErr
	}

	return nil
}

// eliminaDuplicatiStringhe rimuove duplicati da una slice di stringhe.
func eliminaDuplicatiStringhe(lista []string) []string {
	unici := make(map[string]struct{})
	risultato := []string{}

	for _, elemento := range lista {
		if _, presente := unici[elemento]; !presente {
			unici[elemento] = struct{}{}
			risultato = append(risultato, elemento)
		}
	}

	return risultato
}

// executeQueryListEnabledWithBreaker esegue una query SQL utilizzando un circuit breaker.
func executeQueryListEnabledWithBreaker(breaker *gobreaker.CircuitBreaker, db *sql.DB, email string) (string, error) {
	var id string

	// Esegui la query utilizzando il circuit breaker
	_, execErr := breaker.Execute(func() (interface{}, error) {
		// Esegui la query SQL
		rows, err := db.Query("SELECT id_tg FROM users WHERE email='" + email + "'")
		if err != nil {
			return "", err
		}
		defer rows.Close()

		// Itera sui risultati della query
		for rows.Next() {
			// Scansione del risultato nella variabile 'id'
			err := rows.Scan(&id)
			if err != nil {
				return "", err
			}
		}

		// Gestisci eventuali errori dopo rows.Next()
		if err := rows.Err(); err != nil {
			return "", err
		}

		return id, nil
	})

	// Gestisci errori di esecuzione del circuit breaker
	if execErr != nil {
		// Se il circuito è aperto, execErr sarà gobreaker.ErrOpenState
		return "nil", execErr
	}

	return id, nil
}

// executeQueryRowWithBreaker esegue una query SQL utilizzando un circuit breaker.
func executeQueryRowWithBreaker(breaker *gobreaker.CircuitBreaker, db *sql.DB, dati *strutture.Utenti) (*strutture.Utenti, error) {
	// Crea una nuova istanza di Utenti
	utente := strutture.NewUtenti()

	// Variabili temporanee per memorizzare i risultati della query
	var tempn, tempc, tempe, tempp, tempi, tempa string
	var err error

	// Esegui la query utilizzando il circuit breaker
	_, execErr := breaker.Execute(func() (interface{}, error) {
		// Esegui la query SQL con i dati forniti
		err = db.QueryRow("SELECT * FROM users WHERE Email='"+dati.GetEmail()+"' AND Password='"+dati.GetPassword()+"'").
			Scan(&tempn, &tempc, &tempe, &tempp, &tempi, &tempa)

		// Imposta i dati nella struttura Utenti
		utente.SetNome(tempn)
		utente.SetCognome(tempc)
		utente.SetEmail(tempe)
		utente.SetPassword(tempp)
		utente.SetIdTg(tempi)
		utente.SetActive(tempa)

		// Gestisci il caso in cui la query non restituisce alcuna riga
		if isEmptyError(err) {
			log.Println("Nessuna riga trovata.")
			// Imposta i valori di default o "0" nella struttura Utenti
			utente.SetNome("0")
			utente.SetCognome("0")
			utente.SetEmail("0")
			utente.SetPassword("0")
			utente.SetIdTg("0")
			utente.SetActive("0")
			return utente, nil
		}

		// Gestisci altri tipi di errori
		if err != nil {
			return nil, err
		}

		// Restituisci la struttura Utenti popolata con i dati della query
		return utente, nil
	})

	// Gestisci eventuali errori di esecuzione del circuit breaker
	if execErr != nil {
		// Se il circuito è aperto, execErr sarà gobreaker.ErrOpenState
		return utente, execErr
	}

	// Restituisci la struttura Utenti e l'eventuale errore
	return utente, nil
}

// dsn costruisce e restituisce una stringa di connessione al database MySQL utilizzando i parametri configurati.
func dsn(dbName string) string {
	// Ottieni i parametri di configurazione per la connessione al database
	username, err := config.GetParametroFromConfig("databaseuser")
	password, err1 := config.GetParametroFromConfig("databasepassword")
	host, err2 := config.GetParametroFromConfig("databasehost")
	port, err3 := config.GetParametroFromConfig("databaseport")
	hostname := fmt.Sprintf("%s:%s", host, port)

	// Gestisci gli errori durante il recupero dei parametri di configurazione
	if err != nil {
		log.Println(err)
		return ""
	}
	if err1 != nil {
		log.Println(err1)
		return ""
	}
	if err2 != nil {
		log.Println(err2)
		return ""
	}
	if err3 != nil {
		log.Println(err3)
		return ""
	}

	// Costruisci e restituisci la stringa di connessione al database
	return fmt.Sprintf("%s:%s@tcp(%s)/%s", username, password, hostname, dbName)
}

// openDBWithBreaker apre una connessione al database utilizzando un circuit breaker per la gestione degli errori.
func openDBWithBreaker(dsn string) (*sql.DB, error) {
	var db *sql.DB
	var err error

	// Esegue l'apertura del database utilizzando un circuit breaker
	result, execErr := breaker.Execute(func() (interface{}, error) {
		// Apre una connessione al database
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			log.Printf("Errore %s durante l'apertura del database\n", err)
			return nil, err
		}

		// Crea il database se non esiste
		ctx, cancelfunc := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancelfunc()

		_, err = db.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS "+dbname)
		if err != nil {
			log.Printf("Errore %s durante la creazione del database\n", err)
			return nil, err
		}

		return db, nil
	})

	// Se il circuit breaker è aperto, restituisci un errore
	if execErr != nil {
		log.Printf("Errore nel Circuit Breaker : %v\n", execErr)
		return nil, execErr
	}

	return result.(*sql.DB), err
}

func StartDBUtenti() {

	// Passo 1: Connessione al database principale
	_, err := connectDB("")

	if err != nil {
		log.Printf("Errore %s ", err)
	}

	log.Println("Passo 1 completato")

	// Passo 2: Connessione al database specifico per le rotte
	db, err := connectDB(dbname)

	if err != nil {
		log.Printf("Errore %s ", err)
	}

	// Creazione della tabella "users" se non esiste
	err = queryDB(db, "CREATE TABLE IF NOT EXISTS users (name VARCHAR(255),Cognome VARCHAR(255),Email VARCHAR(255),Password VARCHAR(255),id_tg VARCHAR(255) DEFAULT '',active TINYINT,PRIMARY KEY (Email))")

	if err != nil {
		log.Printf("Errore %s ", err)
	}

	log.Println("Passo 2 completato")

	if err != nil {
		log.Printf("Error %s ", err)

	}

	if db != nil {
		defer db.Close()
	}

}

func CreateUser(dati *strutture.Utenti) (*strutture.Utenti, error) {

	db, err := connectDB(dbname)

	if err != nil {
		log.Printf("Errore %s ", err)
		return nil, err
	}
	if db != nil {
		defer db.Close()
	}

	db.SetConnMaxLifetime(time.Minute * 5)
	log.Printf("Connected to DB %s successfully\n", dbname)
	// Inserimento dati nella tabella "users"
	err = queryDB(db, fmt.Sprintf("INSERT INTO users (name, Cognome, Email,Password,id_tg,active) VALUES ('"+dati.GetNome()+"','"+dati.GetCognome()+"','"+dati.GetEmail()+"','"+dati.GetPassword()+"','"+dati.GetIdTg()+"','1')"))

	if err != nil {
		log.Printf("Errore %s ", err)
		return nil, err
	}

	return dati, nil
}

func ReadeUser(dati *strutture.Utenti) (*strutture.Utenti, error) {

	db, err := connectDB(dbname)

	if err != nil {
		log.Printf("Errore %s ", err)
		return nil, err
	}
	if db != nil {
		defer db.Close()
	}

	db.SetConnMaxLifetime(time.Minute * 5)
	log.Printf("Connected to DB %s successfully\n", dbname)
	// Lettura dati nella tabella "users" con circuit breaker
	utente, err := executeQueryRowWithBreaker(breaker, db, dati)

	for breaker.State().String() == gobreaker.StateClosed.String() && err != nil && !isDuplicateKeyError(err) {

		utente, err = executeQueryRowWithBreaker(breaker, db, dati)

	}
	if breaker.State().String() != gobreaker.StateClosed.String() {

		fmt.Println("Stato Aperto " + breaker.Name())

		log.Printf("Error %s", err)
		return nil, err
	}

	if err != nil {
		log.Printf("Error %s ", err)
		return nil, err
	}
	// Se login è stato effettuato sulla piattaforma Telegram aggiorna id_tg
	if dati.GetIdTg() != "nullo" && utente.GetIdTg() == "nullo" {

		UpdateUserTg(dati, dati.GetIdTg())
	}

	if err != nil {
		log.Printf("Error %s when creating DB\n", err)

		return nil, err
	}

	return utente, nil

}

func DeleteUser(dati *strutture.Utenti) (*string, error) {
	a := "ok"

	db, err := connectDB(dbname)

	if err != nil {
		log.Printf("Errore %s ", err)
		return nil, err
	}
	if db != nil {
		defer db.Close()
	}

	db.SetConnMaxLifetime(time.Minute * 5)
	log.Printf("Connected to DB %s successfully\n", dbname)
	// Eliminazione dati nella tabella "users" con circuit breaker
	err = queryDB(db, "DELETE FROM users WHERE Email='"+dati.GetEmail()+"'and Password='"+dati.GetPassword()+"'")

	if err != nil {
		log.Printf("Errore %s ", err)
		return nil, err
	}

	return &a, nil
}

func UpdateUserTg(dati *strutture.Utenti, temp string) error {

	db, err := connectDB(dbname)

	if err != nil {
		log.Printf("Errore %s ", err)
		return err
	}
	if db != nil {
		defer db.Close()
	}

	db.SetConnMaxLifetime(time.Minute * 5)
	log.Printf("Connected to DB %s successfully\n", dbname)
	// Aggiornamento id_tg nella tabella "users" con circuit breaker
	err = queryDB(db, fmt.Sprintf("UPDATE users SET id_tg='"+temp+"' WHERE Email='"+dati.GetEmail()+"' AND id_tg='nullo'"))

	if err != nil {
		log.Printf("Errore %s ", err)
		return err
	}

	return nil
}

func UpdateUser(dati *strutture.Utenti, datiold *strutture.Utenti) (*strutture.Utenti, error) {

	db, err := connectDB(dbname)

	if err != nil {
		log.Printf("Errore %s ", err)
		return nil, err
	}

	if db != nil {
		defer db.Close()
	}
	dati.ToString()
	datiold.ToString()
	db.SetConnMaxLifetime(time.Minute * 5)
	log.Printf("Connected to DB %s successfully\n", dbname)
	// Aggiornamento dati nella tabella "users" con circuit breaker
	err = queryDB(db, fmt.Sprintf("UPDATE users SET name='"+dati.GetNome()+"', Cognome='"+dati.GetCognome()+"',Email='"+dati.GetEmail()+"',Password='"+dati.GetPassword()+"' WHERE Email='"+datiold.GetEmail()+"' AND Password='"+datiold.GetPassword()+"'"))

	if err != nil {
		log.Printf("Errore %s ", err)
		return nil, err
	}

	return dati, nil

}

func GetEmailRoute(lista []string) ([]string, error) {

	// Crea un vettore per immagazzinare i risultati
	var route []string
	var id string

	list := eliminaDuplicatiStringhe(lista)

	db, err := connectDB(dbname)

	if err != nil {
		log.Printf("Errore %s ", err)
		return nil, err
	}
	if db != nil {
		defer db.Close()
	}

	db.SetConnMaxLifetime(time.Minute * 5)
	log.Printf("Connected to DB %s successfully\n", dbname)

	for _, email := range list {

		id, err = executeQueryListEnabledWithBreaker(breaker, db, email)

		for breaker.State().String() == gobreaker.StateClosed.String() && err != nil && !isDuplicateKeyError(err) {

			// Perform your task here
			id, err = executeQueryListEnabledWithBreaker(breaker, db, email)

		}
		if breaker.State().String() != gobreaker.StateClosed.String() {
			// Sleep or add a delay before the next iteration
			fmt.Println("Stato Aperto " + breaker.Name())
			log.Printf("Error %s : %s ", err, "Errore Connessione")
			return nil, err
		}
		route = append(route, id)

	}
	if route == nil {
		err = errors.New("lista vuota")
		log.Printf("Error %s", err)
		return nil, err
	}

	return route, nil
}
