import json
import requests
from circuit_breaker import CircuitBreaker

# Creazione di un'istanza del Circuit Breaker
circuit_breaker = CircuitBreaker()


# Funzione per caricare la configurazione dal file JSON
def load_config(path):
    with open(path, 'r') as file:
        file_content = file.read()

    data = json.loads(file_content)

    return data


# Decoratore del Circuit Breaker per gestire le richieste POST
@circuit_breaker.decorator_request
def handle_request_post(address, route, payload):
    return requests.post(f"http://{address}/api/{route}", payload,
                         headers={"Content-Type": "application/json"})


# Classe per la connessione al server Flask
class ConnectionServer:
    def __init__(self, email, psw, chat_id):
        self.email = email
        self.psw = psw
        self.chat_id = str(chat_id)
        self.config = load_config("config.json")

    def connection_to_server(self):
        # Impostazione dell'URL per inviare i dati inseriti dall'utente al server Flask
        host = "localhost" if self.config["flask_server"] == "" else self.config["flask_server"]
        port = "8888" if self.config["flask_port"] == "" else self.config["flask_port"]
        address = f"{host}:{port}"

        # Creazione del payload da inviare al server
        user_data = {
            "Email": self.email,
            "Password": self.psw,
            "Id_tg": self.chat_id
        }

        # I dati vengono salvati in un dizionario il quale viene trasformato in JSON
        message = json.dumps(user_data)

        # Invio della richiesta POST al server
        response, error = handle_request_post(address, "send", message)
        if response is None:
            return error

        print(response.text)

        # Parsing della risposta JSON del server
        resp = json.loads(response.text)
        return resp

    def logout(self):
        # Impostazione dell'URL per il logout
        host = "localhost" if self.config["flask_server"] == "" else self.config["flask_server"]
        port = "8888" if self.config["flask_port"] == "" else self.config["flask_port"]
        address = f"{host}:{port}"

        # Creazione del payload per il logout
        data = {
            "Email": self.email,
            "Id_tg": self.chat_id
        }
        message = json.dumps(data)

        # Invio della richiesta POST per il logout al server
        response, error = handle_request_post(address, "logout", message)

        if response is None:
            return error

        print(response.text)
        resp = response.text
        return resp
