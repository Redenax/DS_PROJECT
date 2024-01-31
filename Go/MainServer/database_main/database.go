package database

import (
	config "MainServer/config"
	"MainServer/strutture"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/sony/gobreaker"
)

const (
	dbname1            = "routes"
	dbname2            = "slaDB"
	circuitBreakerName = "databaseCircuitBreaker"
)

var breaker *gobreaker.CircuitBreaker

func init() {
	// Imposta l'output dei log su Stdout
	log.SetOutput(os.Stdout)

	// Inizializza il circuit breaker utilizzando le impostazioni ottenute da utility.GetBreakerSettings
	breaker = config.GetBreakerSettings(circuitBreakerName)
}

// isDuplicateKeyError verifica se l'errore è una violazione di chiave duplicata MySQL.
func isDuplicateKeyError(err error) bool {
	mysqlError, ok := err.(*mysql.MySQLError)
	if ok && mysqlError.Number == 1062 {
		// Codice di errore MySQL per violazione di chiave duplicata
		return true
	}
	return false
}

func getListDatabase(db *sql.DB, lista []string, query string) ([]string, error) {

	rows, err := db.Query(query)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	// Itera sui risultati della query
	for rows.Next() {
		var nome string
		err := rows.Scan(&nome)
		if err != nil {
			log.Fatal(err)
		}
		lista = append(lista, nome)
	}
	// Gestisci eventuali errori dopo rows.Next()
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	return lista, nil
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

// connectDB apre una connessione al database utilizzando un circuit breaker per la gestione degli errori.
func connectDBMetrics(name string) (*sql.DB, error) {
	// Apre una connessione al database utilizzando un circuit breaker
	db, err := openDBMetricsWithBreaker(dsnMetrics(name))

	// Riprova a connettersi mentre il circuit breaker è chiuso e si verifica un errore
	for breaker.State().String() == gobreaker.StateClosed.String() && err != nil {
		// Riprova a connettersi
		db, err = openDBWithBreaker(dsnMetrics(name))
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

// queryListDB esegue una query sul database e restituisce una lista di stringhe utilizzando un circuit breaker per la gestione degli errori.
func queryListDB(db *sql.DB, query string) ([]string, error) {
	// Esegue la query e ottiene la lista di stringhe utilizzando un circuit breaker
	list, err := executeQueryListWithBreaker(db, query)

	// Riprova a eseguire la query mentre il circuit breaker è chiuso e si verifica un errore
	for breaker.State().String() == gobreaker.StateClosed.String() && err != nil {
		// Riprova a eseguire la query
		list, err = executeQueryListWithBreaker(db, query)
	}

	// Se il circuit breaker è aperto, restituisci un errore
	if breaker.State().String() != gobreaker.StateClosed.String() {
		log.Println("Stato Aperto " + breaker.Name())
		log.Printf("Error %s ", err)
		return nil, err
	}

	return list, nil
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

// executeQueryListWithBreaker esegue una query sul database e restituisce una lista di stringhe utilizzando un circuit breaker per la gestione degli errori.
func executeQueryListWithBreaker(db *sql.DB, query string) ([]string, error) {
	// Inizializza una lista vuota
	lista := make([]string, 0, 5)
	var err1 error

	// Esegue la query utilizzando un circuit breaker
	_, execErr := breaker.Execute(func() (interface{}, error) {
		// Ottiene la lista dal database utilizzando una funzione di utilità
		lista, err1 = getListDatabase(db, lista, query)
		if err1 != nil {
			return nil, err1
		}

		return lista, nil
	})

	// Se il circuit breaker è aperto, restituisci un errore
	if execErr != nil {
		return nil, execErr
	}

	return lista, nil
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

		_, err = db.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS "+dbname1)
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

func openDBMetricsWithBreaker(dsn string) (*sql.DB, error) {
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

		_, err = db.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS "+dbname2)
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

func dsnMetrics(dbName string) string {
	// Ottieni i parametri di configurazione per la connessione al database
	username, err := config.GetParametroFromConfig("databasemetricsuser")
	password, err1 := config.GetParametroFromConfig("databasemetricspassword")
	host, err2 := config.GetParametroFromConfig("databasemetricshost")
	port, err3 := config.GetParametroFromConfig("databasemetricsport")
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

// StartDBRoute inizializza il database e crea le tabelle necessarie per le province e le rotte.
func StartDBRoute(province []string) {
	// Passo 1: Connessione al database principale
	_, err := connectDB("")

	if err != nil {
		log.Printf("Errore %s ", err)
	}

	log.Println("Passo 1 completato")

	// Passo 2: Connessione al database specifico per le rotte
	db, err := connectDB(dbname1)

	if err != nil {
		log.Printf("Errore %s ", err)
	}

	// Creazione della tabella "province" se non esiste
	err = queryDB(db, "CREATE TABLE IF NOT EXISTS province (id INT NOT NULL AUTO_INCREMENT, name VARCHAR(255), PRIMARY KEY (id))")

	if err != nil {
		log.Printf("Errore %s ", err)
	}

	log.Println("Passo 2 completato")

	// Passo 3: Inserimento delle province nel database
	for i := range province {
		err = queryDB(db, fmt.Sprintf("INSERT INTO province (name) SELECT * FROM (SELECT '"+province[i]+"') AS tmp WHERE NOT EXISTS (SELECT name FROM province WHERE name = '"+province[i]+"') LIMIT 1;"))
	}

	if err != nil {
		log.Printf("Errore %s ", err)
	}

	log.Println("Passo 3 completato")

	// Creazione della tabella "routes" se non esiste
	err = queryDB(db, "CREATE TABLE IF NOT EXISTS routes (name VARCHAR(255), partenza int, destinazione int, email varchar(255), notify tinyint, PRIMARY KEY (partenza, destinazione, email))")

	if err != nil {
		log.Printf("Errore %s ", err)
	}

	// Chiusura della connessione al database
	if db != nil {
		db.Close()
	}
	_, err = connectDBMetrics("")

	if err != nil {
		log.Printf("Errore %s ", err)
	}
	// Passo 4: Connessione al database specifico per le rotte
	db1, err := connectDBMetrics(dbname2)

	if err != nil {
		log.Printf("Errore %s ", err)
	}

	err = queryDB(db1, "CREATE TABLE IF NOT EXISTS metrics (id INT NOT NULL AUTO_INCREMENT, name VARCHAR(255),viol_max FLOAT,time DATETIME, PRIMARY KEY (id))")

	if err != nil {
		log.Printf("Errore %s ", err)
	}
	log.Println("Passo 4 completato")
	err = queryDB(db1, "CREATE TABLE IF NOT EXISTS serie_temp (id INT NOT NULL AUTO_INCREMENT, name VARCHAR(255),error_std FLOAT,trend_function TEXT, PRIMARY KEY (id))")

	if err != nil {
		log.Printf("Errore %s ", err)
	}
	log.Println("Passo 5 completato")
	if db1 != nil {
		db1.Close()
	}
}

// CreateRoute crea una nuova rotta nel database utilizzando i dati forniti.
func CreateRoute(dati *strutture.Routes) (*string, error) {
	var id_partenza, id_destinazione int

	// Connessione al database
	db, err := connectDB(dbname1)
	if err != nil {
		log.Printf("Errore %s ", err)
		return nil, err
	}
	defer db.Close()

	db.SetConnMaxLifetime(30 * time.Second)
	log.Printf("Connesso al database %s con successo\n", dbname1)

	// Ottieni gli ID delle province di partenza e destinazione dal database
	err = db.QueryRow("SELECT id FROM province WHERE name='" + dati.GetPartenza() + "'").Scan(&id_partenza)
	err1 := db.QueryRow("SELECT id FROM province WHERE name='" + dati.GetDestinazione() + "'").Scan(&id_destinazione)

	// Controlla se le province di partenza e destinazione sono state trovate
	if id_partenza == 0 {
		log.Printf("Partenza non trovata")
		return nil, err
	}

	if id_destinazione == 0 {
		log.Printf("Destinazione non trovata")
		return nil, err1
	}

	// Imposta il nome della rotta utilizzando le province di partenza e destinazione
	dati.SetNome(dati.GetPartenza() + "_" + dati.GetDestinazione())

	// Esegui la query per inserire la nuova rotta nel database
	err = queryDB(db, fmt.Sprintf("INSERT INTO routes (name, partenza, destinazione, email, notify) VALUES ('"+dati.GetNome()+"','"+strconv.Itoa(id_partenza)+"','"+strconv.Itoa(id_destinazione)+"','"+dati.GetEmail()+"','0')"))
	if err != nil {
		log.Printf("Errore %s ", err)
		return nil, err
	}

	// Controlla se c'è un errore di chiave duplicata (rotta già presente nel database)
	if isDuplicateKeyError(err1) {
		err1 := errors.New("già salvato in memoria")
		log.Printf("Errore %s ", err1)
		return nil, err1
	}

	// Restituisci un puntatore a una stringa "ok" se tutto va bene
	a := "ok"
	return &a, nil
}

// DeleteRoute elimina una rotta dal database utilizzando i dati forniti.
func DeleteRoute(dati *strutture.Routes) (*string, error) {
	var id_partenza, id_destinazione int
	var a string

	// Connessione al database
	db, err := connectDB(dbname1)
	if err != nil {
		log.Printf("Errore %s ", err)
		return nil, err
	}
	defer db.Close()

	db.SetConnMaxLifetime(30 * time.Second)
	log.Printf("Connesso al database %s con successo\n", dbname1)

	// Ottieni gli ID delle province di partenza e destinazione dal database
	err = db.QueryRow("SELECT id FROM province WHERE name='" + dati.GetPartenza() + "'").Scan(&id_partenza)
	err1 := db.QueryRow("SELECT id FROM province WHERE name='" + dati.GetDestinazione() + "'").Scan(&id_destinazione)

	// Controlla se le province di partenza e destinazione sono state trovate
	if id_partenza == 0 {
		log.Printf("Partenza non trovata")
		return nil, err
	}

	if id_destinazione == 0 {
		log.Printf("Destinazione non trovata")
		return nil, err1
	}

	// Esegui la query per eliminare la rotta dal database
	err = queryDB(db, fmt.Sprintf("DELETE FROM routes WHERE partenza='"+strconv.Itoa(id_partenza)+"' AND destinazione='"+strconv.Itoa(id_destinazione)+"' AND email='"+dati.GetEmail()+"'"))
	if err != nil {
		log.Printf("Errore %s ", err)
		return nil, err
	}

	// Restituisci un puntatore a una stringa "ok" se tutto va bene
	a = "ok"
	return &a, nil
}

// EnableRoutes abilita le notifiche per tutte le rotte associate all'email fornita.
func EnableRoutes(route *strutture.Routes) ([]string, error) {
	// Crea un vettore per immagazzinare i risultati
	var routes []string
	var db *sql.DB
	var err error

	// Connessione al database
	db, err = connectDB(dbname1)
	if err != nil {
		log.Printf("Errore %s ", err)
		return nil, err
	}
	defer db.Close()

	db.SetConnMaxLifetime(30 * time.Second)
	log.Printf("Connesso al database %s con successo\n", dbname1)

	// Esegui la query per abilitare le notifiche per tutte le rotte associate all'email
	err = queryDB(db, fmt.Sprintf("UPDATE routes SET notify='%d' WHERE email='"+route.GetEmail()+"'", 1))
	if err != nil {
		log.Printf("Errore %s ", err)
		return nil, err
	}

	// Ottieni la lista aggiornata delle rotte per l'email fornita
	routes, err = GetRoute(route.GetEmail())
	if err != nil {
		log.Printf("Errore %s ", err)
		return nil, err
	}

	// Controlla se la lista è vuota
	if routes == nil {
		err = errors.New("lista vuota")
		log.Printf("Errore %s", err)
	}

	return routes, nil
}

// DisableRoutes disabilita le notifiche per tutte le rotte associate all'email fornita.
func DisableRoutes(route *strutture.Routes) (string, error) {
	// "a" viene utilizzato solo come valore di ritorno, quindi è dichiarato direttamente
	a := "ok"
	db, err := connectDB(dbname1)

	if err != nil {
		log.Printf("Errore %s ", err)
		return "", err
	}
	defer db.Close()

	db.SetConnMaxLifetime(30 * time.Second)
	log.Printf("Connesso al database %s con successo\n", dbname1)

	// Esegui la query per disabilitare le notifiche per tutte le rotte associate all'email
	err = queryDB(db, fmt.Sprintf("UPDATE routes SET notify='%d' WHERE email='"+route.GetEmail()+"'", 0))
	if err != nil {
		log.Printf("Errore %s ", err)
		return "", err
	}

	return a, nil
}

// GetProvince restituisce una lista di province dal database.
func GetProvince() ([]string, error) {
	// Crea un vettore per immagazzinare i risultati
	var province []string
	var db *sql.DB
	var err error

	// Connessione al database
	db, err = connectDB(dbname1)
	if err != nil {
		log.Printf("Errore %s ", err)
		return nil, err
	}
	defer db.Close()

	db.SetConnMaxLifetime(30 * time.Second)
	log.Printf("Connesso al database %s con successo\n", dbname1)

	// Esegui la query per ottenere la lista delle province
	province, err = queryListDB(db, "SELECT name FROM province")
	if err != nil {
		log.Printf("Errore %s ", err)
		return nil, err
	}

	// Controlla se la lista è vuota
	if province == nil {
		err = errors.New("lista vuota")
		log.Printf("Errore %s", err)
	}

	return province, nil
}

// GetRoute restituisce una lista di rotte associate all'email fornita dal database.
func GetRoute(email string) ([]string, error) {
	// Crea un vettore per immagazzinare i risultati
	var route []string
	var db *sql.DB
	var err error

	// Connessione al database
	db, err = connectDB(dbname1)
	if err != nil {
		log.Printf("Errore %s ", err)
		return nil, err
	}
	defer db.Close()

	db.SetConnMaxLifetime(30 * time.Second)
	log.Printf("Connesso al database %s con successo\n", dbname1)

	// Esegui la query per ottenere la lista delle rotte associate all'email
	route, err = queryListDB(db, fmt.Sprintf("SELECT name FROM routes WHERE email='%s'", email))
	if err != nil {
		log.Printf("Errore %s ", err)
		return nil, err
	}

	// Controlla se la lista è vuota
	if route == nil {
		err = errors.New("lista vuota")
		log.Printf("Errore %s", err)
	}

	return route, nil
}

// GetEmailRoute restituisce una lista di email associate alle rotte con notifiche abilitate dal database.
func GetEmailRoute() ([]string, error) {
	// Crea un vettore per immagazzinare i risultati
	var route []string
	var db *sql.DB
	var err error

	// Connessione al database
	db, err = connectDB(dbname1)
	if err != nil {
		log.Printf("Errore %s ", err)
		return nil, err
	}
	defer db.Close()

	db.SetConnMaxLifetime(30 * time.Second)
	log.Printf("Connesso al database %s con successo\n", dbname1)

	// Esegui la query per ottenere la lista di email associate alle rotte con notifiche abilitate
	route, err = queryListDB(db, "SELECT email FROM routes WHERE notify='1'")
	if err != nil {
		log.Printf("Errore %s ", err)
		return nil, err
	}

	// Controlla se la lista è vuota
	if route == nil {
		err = errors.New("lista vuota")
		log.Printf("Errore %s", err)
	}

	return route, nil
}

// DeleteRoute elimina una rotta dal database utilizzando i dati forniti.
func DeleteRouteEmail(dati *strutture.Authentication) (*string, error) {

	var a string

	// Connessione al database
	db, err := connectDB(dbname1)
	if err != nil {
		log.Printf("Errore %s ", err)
		return nil, err
	}
	defer db.Close()

	db.SetConnMaxLifetime(30 * time.Second)
	log.Printf("Connesso al database %s con successo\n", dbname1)

	// Esegui la query per eliminare la rotta dal database
	err = queryDB(db, fmt.Sprintf("DELETE FROM routes WHERE email='"+dati.GetEmail()+"'"))
	if err != nil {
		log.Printf("Errore %s ", err)
		return nil, err
	}

	// Restituisci un puntatore a una stringa "ok" se tutto va bene
	a = "ok"
	return &a, nil
}
