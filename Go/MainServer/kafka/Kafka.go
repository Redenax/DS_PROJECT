package kafka

import (
	config "MainServer/config"
	"context"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/segmentio/kafka-go"
	"github.com/sony/gobreaker"
)

var breaker *gobreaker.CircuitBreaker

func init() {
	log.SetOutput(os.Stdout)
	// Inizializza il circuit breaker
	breaker = config.GetBreakerSettings("BreakerConnectionKafka")
}

func executeConnectionBreaker(breaker *gobreaker.CircuitBreaker, brokerAddress string) (*kafka.Conn, error) {
	var admin *kafka.Conn
	var err error
	// Use gobreaker.Execute to handle circuit breaking
	_, execErr := breaker.Execute(func() (interface{}, error) {
		// Inside the Execute function, create a new Kafka connection
		admin, err = kafka.DialContext(context.Background(), "tcp", brokerAddress)
		if err != nil {
			log.Printf("Error connecting to Kafka: %v", err)
			return nil, err
		}

		return admin, nil
	})

	if execErr != nil {
		// If the circuit is open, execErr will be gobreaker.ErrOpenState
		return nil, execErr
	}

	return admin, nil
}
func GetKafkaAddress() string {

	brokerAddress, err := config.GetParametroFromConfig("kafkabrokeraddress")

	if err != nil {
		log.Println(err)
		return ""
	}

	return brokerAddress
}

func KafkaProducer(topic string, messaggio []byte) {
	brokerAddress := GetKafkaAddress()

	admin, err := executeConnectionBreaker(breaker, brokerAddress)
	// Chiama la funzione openDBWithBreaker
	for breaker.State().String() == gobreaker.StateClosed.String() && err != nil {
		// Perform your task here
		admin, err = executeConnectionBreaker(breaker, brokerAddress)
	}

	if breaker.State().String() != gobreaker.StateClosed.String() {
		// Sleep or add a delay before the next iteration
		log.Println("Stato Aperto " + breaker.Name())

		log.Printf("Error %s ", err)
		return
	}

	if admin != nil {
		defer admin.Close()
	}
	if err != nil {
		log.Printf("Error %s ", err)
		return
	}

	// Create a new Kafka writer
	writer := kafka.NewWriter(kafka.WriterConfig{
		Brokers:  []string{brokerAddress},
		Topic:    topic,
		Balancer: &kafka.LeastBytes{},
	})

	// Produce a message to the topic
	message := kafka.Message{
		Value: messaggio,
	}

	err = writer.WriteMessages(context.Background(), message)
	if err != nil {
		log.Printf("Errore nella scrittura : %v\n", err)
		return
	}

	log.Println("Messaggio inviato al topic: " + string(messaggio))

	// Close the writer
	err = writer.Close()
	if err != nil {
		log.Printf("Errore chiusura Kafka writer: %v\n", err)
		return
	}

}
