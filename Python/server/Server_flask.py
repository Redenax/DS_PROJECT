import asyncio
import json
import threading
import time
from threading import Thread
from queue import Queue
import telegram
from prometheus_client import generate_latest, REGISTRY, CONTENT_TYPE_LATEST, Gauge
from telegram import KeyboardButton, ReplyKeyboardMarkup
from telegram.error import TelegramError
from flask import Flask, request, jsonify
import requests
from circuit_breaker import CircuitBreaker
from kafka.consumer_kafka import KafkaConsumer

# Inizializzazione dell'app Flask
app = Flask(__name__)

# Inizializzazione di variabili globali
consumer = None
queue_resp = Queue()
firstRun = True

# Definizione di metriche Prometheus
message_sent = Gauge('Message_sent',
                     'numero di messaggi inviati agli utenti connessi')
message_sent.set(0)

request_time_to_main = Gauge('request_time_to_main',
                             'Istogramma per definire il tempo di richiesta tra il flask e main server')
request_time_to_main.set(0)

circuit_breaker = CircuitBreaker(request_time_to_main)


# Decoratore per gestire richieste POST con il Circuit Breaker
@circuit_breaker.decorator_request
def handle_request_post(route, payload):
    return requests.post(f"http://{main_address}/api/v1/{route}", payload,
                         headers={"Content-Type": "application/json"})


# Decoratori per gestire richieste GET con il Circuit Breaker
@circuit_breaker.decorator_request
def handle_request_get(route):
    return requests.get(f"http://{main_address}/api/v1/{route}")


# Funzione per eseguire l'accesso in un thread separato
def thread_handle_login(data, queue_response):
    print(f"Received data as dictionary: {data}")

    # Estrai l'indirizzo email dai dati
    email = data['Email']
    id_tg = data['Id_tg']
    # Converti i dati in formato JSON
    payload = json.dumps(data)

    # Esegue la richiesta di autenticazione
    resp, error = handle_request_post("authentication", payload)

    # Se la risposta è nulla, c'è stato un errore durante l'autenticazione
    if resp is None:
        # Creare una risposta con il messaggio di errore e metterla nella coda di risposta
        response = {"Authorization": error}
        queue_response.put(response)
        return

    # Creare dati per la richiesta di abilitazione delle route
    route_data = {"Email": email}

    # Converti i dati delle route in formato JSON
    payload = json.dumps(route_data)

    # Esegue la richiesta di abilitazione delle route
    route_request, error = handle_request_post("enableRoute", payload)

    # Se la richiesta delle route è nulla, c'è stato un errore durante l'abilitazione delle route
    if route_request is None:
        # Creare una risposta con il messaggio di errore e metterla nella coda di risposta
        response = {"Authorization": error}
        queue_response.put(response)
        return

    # Estrai le route dalla risposta e crea una risposta completa
    routes = route_request.json()

    response = {
        "Authorization": resp.text,
        "route_list": routes,
        "Id_tg": id_tg
    }

    # Metti la risposta nella coda di risposta
    queue_response.put(response)


# Gestione della richiesta di login tramite Flask
@app.route('/api/send', methods=['POST'])
def handle_login():
    # Ricevo i dati in formato JSON dalla richiesta POST
    global firstRun
    data = request.json
    print(data)

    # Lancio un nuovo thread per gestire l'accesso dell'utente
    thread = Thread(target=thread_handle_login, args=(data, queue_resp))
    thread.start()

    # Utilizzo un blocco di thread per garantire la sincronizzazione
    with threading.Lock():
        # Ottengo la risposta dal thread di gestione dell'accesso
        response = queue_resp.get()

        if response['Authorization'] == "Authorized":
            # Esegue il login tramite Kafka
            consumer.set_chat(response['Id_tg'], response['route_list'])
            if firstRun:
                firstRun = False
                # Avvio del consumatore Kafka
                consumer.start_consumer()

            # Restituisco la risposta di autorizzazione come JSON
            return jsonify(response['Authorization'])
        else:
            # Restituisco la risposta di errore come JSON
            return jsonify(response['Authorization'])


# Funzione per eseguire il logout tramite Flask
@app.route('/api/logout', methods=['POST'])
def handle_logout():
    global main_address

    print(request.data)

    # Ottiene i dati JSON dalla richiesta POST
    data = request.json
    email = {"Email": data['Email']}
    id_tg = data['Id_tg']
    payload = json.dumps(email)

    # Esegue la richiesta di disabilitazione della route tramite una funzione esterna
    route, error = handle_request_post("disableRoute", payload)

    if route is None:
        # Se la richiesta di disabilitazione della route non ha avuto successo, restituisce l'errore come JSON
        return jsonify(error)

    # Esegue il logout tramite Kafka
    consumer.logout_command(id_tg)
    # Restituisce un messaggio JSON indicando che il logout è stato effettuato con successo
    return "logout effettuato"


@app.route('/metrics')
def metrics():
    return generate_latest(REGISTRY), 200, {'Content-Type': CONTENT_TYPE_LATEST}


# Funzione per inviare notifiche agli utenti
def user_advertise():
    # Esegue una richiesta GET per ottenere la lista degli utenti Telegram registrati
    response, error = handle_request_get("gettg")

    if response is None:
        # Se la risposta non è valida, esce dalla funzione
        return

    # Verifica se la risposta contiene utenti Telegram (non vuota)
    if response.text.lower().strip() not in ["lista vuota", "lista vuota\n"]:
        # Ottiene la lista degli ID degli utenti Telegram dalla risposta JSON
        chat_ids = response.json()

        # Itera attraverso gli ID degli utenti Telegram
        for chat_id in chat_ids:
            # Configura una tastiera di login per il messaggio
            login_key = [
                [KeyboardButton("/login")],
            ]
            login_markup = ReplyKeyboardMarkup(login_key, resize_keyboard=True)

            # Messaggio da inviare agli utenti
            tg_msg = "Il server notification si è dovuto riavviare dopo un crash inaspettato\nPer favore effettuare " \
                     "il login!"
            # Crea un oggetto Bot di Telegram
            bot = telegram.Bot(api_bot)
            try:
                # Invia il messaggio agli utenti con la tastiera di login
                asyncio.run(bot.send_message(chat_id, text=tg_msg, reply_markup=login_markup))
            except TelegramError as err:
                # Gestisce gli errori durante l'invio del messaggio
                print(f"Errore durante l'invio del messaggio{err}")


# Punto d'ingresso dell'app Flask
if __name__ == '__main__':
    global main_address
    global kafka_address
    global api_bot

    # Caricamento delle configurazioni dal file JSON
    with open("config.json", 'r') as file:
        file_content = file.read()

    data = json.loads(file_content)

    # Configurazioni per l'host e la porta di Flask
    flask_host = "localhost" if data["flask_server"] == "" else data["flask_server"]
    flask_port = "8888" if data["flask_port"] == "" else data["flask_port"]

    # Configurazioni per l'host e la porta del server principale
    main_address = "localhost" if data["server_main_host"] == "" else data["server_main_host"]
    port_main = "25536" if data["server_main_port"] == "" else data["server_main_port"]

    # Configurazioni per l'host e la porta di Kafka
    kafka_host = "localhost" if data["kafka_host"] == "" else data["kafka_host"]
    kafka_port = "9093" if data["kafka_port"] == "" else data["kafka_port"]

    # API Token del bot Telegram
    api_bot = data["api_bot"]

    # Composizione degli indirizzi
    main_address = main_address + ':' + port_main
    kafka_address = kafka_host + ':' + kafka_port

    time.sleep(70)

    # Invio di notifiche agli utenti che erano online prima del riavvio
    user_advertise()

    # Creazione di un oggetto KafkaConsumer con le informazioni fornite
    consumer = KafkaConsumer(kafka_address, api_bot, message_sent)

    # Avvio dell'app Flask
    app.run(flask_host, flask_port, debug=True, use_reloader=False)
