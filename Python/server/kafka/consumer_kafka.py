import asyncio
import json
import threading
import time

import telegram
from confluent_kafka import Consumer, KafkaError

# Variabile di controllo per il consumatore periodico
keep_running = False


class KafkaConsumer:

    def __init__(self, broker, api_bot, message_sent):
        self.chat_id = None
        self.broker = broker
        self.api_bot = api_bot
        self.message_sent = message_sent

    async def handle_kafka_message(self, message):
        global keep_running
        msg_count = 0
        # Gestisci i messaggi da Kafka qui solo se keep running è true
        if keep_running:
            if self.chat_id is not None:
                bot = telegram.Bot(self.api_bot)
                for chat, topics in self.chat_id.items():
                    for topic in topics:
                        if topic in message[topic]:
                            msg = message[topic]

                            # Invia il messaggio Telegram utilizzando l'API del bot
                            await bot.send_message(chat, text=msg)
                            time.sleep(1)
                            msg_count += 1

        self.message_sent.set(msg_count)

    def set_kafka_consumer(self, kafka_config):
        # Crea un consumatore Kafka con le configurazioni specificate
        consumer = Consumer(kafka_config)
        consumer.subscribe(['tratte-auto'])

        try:
            while keep_running:
                # Polling per ricevere messaggi da Kafka
                msg = consumer.poll(30)
                if msg is None:
                    continue
                if msg.error():
                    # Gestione degli errori Kafka
                    if msg.error().code() == KafkaError._PARTITION_EOF:
                        continue
                    else:
                        print(f"Errore Kafka: {msg.error()}")
                        break

                data = msg.value().decode('utf-8')
                message = json.loads(data)
                # Chiamata alla funzione per gestire il messaggio Kafka
                asyncio.run(self.handle_kafka_message(message))

        except KeyboardInterrupt:
            pass
        finally:
            print("annullo iscrizione al topic")
            # Annulla l'iscrizione e chiudi il consumatore Kafka
            consumer.unsubscribe()
            consumer.close()

    def set_chat(self, chat_id, topics):
        self.chat_id = {
            chat_id: topics
        }

    def periodic_kafka_consumer(self):
        global keep_running
        # Configurazione di Kafka
        kafka_config = {
            'bootstrap.servers': self.broker,
            'group.id': 1,
            'auto.offset.reset': 'earliest'
        }
        # Esegui il consumatore Kafka periodicamente ogni volta che keep_running è True
        while keep_running:
            self.set_kafka_consumer(kafka_config)

    def logout_command(self, chat_id):
        # Chiamato quando viene eseguito il comando di logout
        del self.chat_id[chat_id]
        print("Logout command executed.")

    def start_consumer(self):
        global keep_running
        keep_running = True
        # Avvia il thread per il consumatore Kafka periodico
        kafka_thread = threading.Thread(target=self.periodic_kafka_consumer, daemon=True)
        kafka_thread.start()
