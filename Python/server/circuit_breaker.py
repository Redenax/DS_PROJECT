import time
from datetime import datetime
from enum import Enum
import requests


# Definizione di uno stato tramite un'enumerazione
class State(Enum):
    circuit_close = 3
    circuit_half_open = 1
    circuit_open = 0


# Classe per implementare il Circuit Breaker
class CircuitBreaker:
    def __init__(self, request_time):
        self.reset_timeout = 30
        self.circuit_state = State.circuit_close
        self.last_failure_time = None
        self.request_time = request_time

    # Apri il Circuit Breaker
    def open_circuit(self):
        self.circuit_state = State.circuit_open
        self.last_failure_time = time.time()

    # Chiudi il Circuit Breaker
    def close_circuit(self):
        self.circuit_state = State.circuit_close

    # Passa a uno stato semi-aperto
    def half_open_circuit(self):
        self.circuit_state = State.circuit_half_open

    # Verifica se il Circuit Breaker è aperto
    def is_circuit_open(self):
        if self.circuit_state == State.circuit_open:
            current_time = time.time()

            # Verifica se è trascorso il tempo di reset
            if current_time - self.last_failure_time > self.reset_timeout:
                self.half_open_circuit()

                return False
            return True
        return False

    # Gestisce le eccezioni durante la richiesta POST
    def handle_exception(self, error, error_message):
        print(error_message, error)
        return error_message

    # Decoratore per gestire le richieste con il Circuit Breaker
    def decorator_request(self, request):
        def handle_request(*args, **kwargs):
            print("Esecuzione del mio decorator prima della funzione originale.")
            time = datetime.now()
            response = None
            error = None
            if not self.is_circuit_open():
                for i in range(self.circuit_state.value):
                    try:
                        print(i)

                        # Esegue la richiesta originale
                        response = request(*args, **kwargs)

                        time_after = datetime.now()
                        time_difference = time_after - time

                        difference_in_seconds = time_difference.total_seconds()

                        self.request_time.set(difference_in_seconds)

                        # Verifica la risposta e il codice di stato HTTP
                        if response.status_code == 200:

                            if self.circuit_state == State.circuit_half_open:
                                self.close_circuit()

                            return response, error

                        response.raise_for_status()

                    except requests.exceptions.HTTPError as err:
                        # Gestione degli errori HTTP
                        error = self.handle_exception(err, err.response.status_code)
                        response = None
                        return response, error

                    except requests.exceptions.ConnectionError as err:
                        # Gestione degli errori di connessione
                        if i == self.circuit_state.value - 1:
                            self.open_circuit()
                            error = self.handle_exception(err, "Errore di connessione")

                    except requests.exceptions.Timeout as err:
                        # Gestione degli errori di timeout
                        if i == self.circuit_state.value - 1:
                            self.open_circuit()
                            error = self.handle_exception(err, "Timeout della richiesta")

                    except requests.exceptions.RequestException as err:
                        # Gestione generica delle eccezioni durante la richiesta
                        error = self.handle_exception(err, "Errore durante la richiesta")
            else:
                # Gestione del caso in cui il Circuit Breaker è aperto
                error = "Errore di connessione"

            print("Esecuzione del mio decorator dopo la funzione originale.")
            return response, error

        return handle_request
