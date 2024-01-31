package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sony/gobreaker"
)

// Config rappresenta la struttura del file di configurazione
type Config struct {
	Database struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
		DBName   string `json:"dbname"`
	} `json:"database"`
}

func GetParametroFromConfig(parametro string) (string, error) {
	// Ottieni la directory corrente del programma
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("errore nell'ottenere la directory corrente: %v", err)
	}

	// Costruisci il percorso completo al file di configurazione
	configPath := filepath.Join(currentDir, "/config/config.json")

	// Leggi il contenuto del file di configurazione
	configData, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("errore nella lettura del file di configurazione: %v", err)
	}

	// Decodifica il file di configurazione nella struttura config
	var config Config
	err = json.Unmarshal(configData, &config)
	if err != nil {
		return "", fmt.Errorf("errore nella decodifica del file di configurazione: %v", err)
	}

	// Cerca il parametro specificato nella struttura config
	switch strings.ToLower(parametro) {
	case "databasehost":
		if config.Database.Host != "" {
			return config.Database.Host, nil
		}
	case "databaseport":
		if config.Database.Port != 0 {
			return fmt.Sprintf("%d", config.Database.Port), nil
		}
	case "databaseuser":
		if config.Database.User != "" {
			return config.Database.User, nil
		}
	case "databasepassword":
		if config.Database.Password != "" {
			return config.Database.Password, nil
		}
	case "databasedbname":
		if config.Database.DBName != "" {
			return config.Database.DBName, nil
		}
	}

	// Se il valore non è nel file di configurazione, cerca una variabile d'ambiente associata
	envValue := os.Getenv(strings.ToUpper(parametro))
	if envValue != "" {
		return envValue, nil
	}

	// Valore di default per ogni parametro
	defaultValues := map[string]string{
		"databasehost":     "localhost",
		"databaseport":     "3306",
		"databaseuser":     "root",
		"databasepassword": "",
		"databasedbname":   "dbname",
	}

	defaultValue, exists := defaultValues[strings.ToLower(parametro)]
	if exists {
		return defaultValue, nil
	}

	return "", fmt.Errorf("parametro non valido: %s", parametro)
}

func GetBreakerSettings(name string) *gobreaker.CircuitBreaker {
	// Inizializza il circuit breaker
	breaker := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:    name,
		Timeout: 3 * time.Second, // Timeout per ogni chiamata
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Impostare le condizioni per il circuito aperto, se ci sono stati troppi errori
			return counts.ConsecutiveFailures > 3
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.Printf("%s state changed from %s to %s", name, from, to)
		},
	})
	return breaker
}
