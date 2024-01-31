import json
from telegram import Update, KeyboardButton, ReplyKeyboardMarkup, ReplyKeyboardRemove
from telegram.ext import ApplicationBuilder, ContextTypes, CommandHandler, ConversationHandler, MessageHandler, filters
from connectionserver import ConnectionServer

# Definizione degli stati della conversazione
USERNAME, PASSWORD = range(2)

# Numero massimo di tentativi di login
NUM_TRY = 2


# Funzione di avvio del bot
async def start(update: Update, context: ContextTypes.DEFAULT_TYPE):
    # Configurazione della tastiera con il pulsante di login
    keyboard = [
        [KeyboardButton("/login")],
    ]

    reply_markup = ReplyKeyboardMarkup(keyboard, resize_keyboard=True)

    # Invia un messaggio di benvenuto con il pulsante di login
    chat_id = update.effective_chat.id
    await context.bot.send_message(chat_id,
                                   "Ciao " + update.message.chat.first_name + "!! Benvenuto in Traffic bot! Premi "
                                                                              "il comando di login per "
                                                                              "effettuare l'accesso.\n"
                                                                              "Digita /cancel per annullare il login.",
                                   reply_markup=reply_markup)


# Funzione per avviare la conversazione di login
async def login(update: Update, context: ContextTypes.DEFAULT_TYPE):
    reply_markup = ReplyKeyboardRemove()
    await update.message.reply_text("Per favore inserisci l'email per effettuare l'accesso:\n",
                                    reply_markup=reply_markup)
    return USERNAME


# Funzione per gestire l'inserimento dell'email
async def username(update: Update, context: ContextTypes.DEFAULT_TYPE):
    # Estrae l'email dal messaggio dell'utente
    email = update.message.text

    # Verifica se l'email inserita è vuota o non valida
    if email == '':
        await update.message.reply_text("Inserimento non valido, per favore inserisci un'email valida")
        return USERNAME

    # Aggiorna il contesto utente con l'email inserita
    context.user_data['email'] = email

    # Inizializza il conteggio dei tentativi se non esiste
    if 'try' not in context.user_data:
        context.user_data['try'] = 0

    # Chiede all'utente d'inserire la password
    await update.message.reply_text('Inserisci la password:')

    # Passa alla fase successiva della conversazione (PASSWORD)
    return PASSWORD


# Funzione per gestire l'inserimento della password e autenticare l'utente
async def password(update: Update, context: ContextTypes.DEFAULT_TYPE):
    # Estrae l'email dall'user_data del contesto
    email = context.user_data['email']
    # Estrae la password dal messaggio dell'utente
    psw = update.message.text

    # Configurazione della tastiera di login e logout
    login_key = [
        [KeyboardButton("/login")],
    ]
    logout_key = [
        [KeyboardButton("/logout")],
    ]

    login_markup = ReplyKeyboardMarkup(login_key, resize_keyboard=True)

    # Connessione al notification server e gestione delle risposte
    connect = ConnectionServer(email, psw, update.effective_chat.id)
    response = connect.connection_to_server()
    print(response)

    if response == "Authorized":
        # Se l'autenticazione ha successo, aggiorna i dati utente nel contesto
        context.user_data['try'] = 0
        context.user_data['logged'] = True
        context.user_data['connect'] = connect

        # Configura la tastiera per il logout
        logout_markup = ReplyKeyboardMarkup(logout_key, resize_keyboard=True)

        # Invia un messaggio di conferma e la tastiera per il logout
        await update.message.reply_text("Utente trovato, il servizio sta per partire")
        await update.message.reply_text("Per effettuare il logout premi logout", reply_markup=logout_markup)

        # Termina la conversazione
        return ConversationHandler.END

    # Gestione degli errori di connessione
    elif response == "Errore di connessione":
        await update.message.reply_text("Server di autenticazione momentaneamente offline.\n"
                                        "Riprovare piu tardi.", reply_markup=login_markup)
        return ConversationHandler.END
    elif response == "Timeout della richiesta":
        await update.message.reply_text("Timeout della richiesta.\n"
                                        "Riprovare piu tardi.", reply_markup=login_markup)
        return ConversationHandler.END
    elif response == "Errore durante la richiesta":
        await update.message.reply_text("Errore durante la richiesta per favore riprovare.\n"
                                        , reply_markup=login_markup)
        return ConversationHandler.END
    else:
        # Gestione degli errori di login
        await update.message.reply_text("Id non trovato, è possibile che tu abbia inserito un id errato o che ancora "
                                        "non esiste alcun utente con quell'ID.")

        if context.user_data['try'] != NUM_TRY:
            # Se ci sono ancora tentativi rimasti, chiede nuovamente l'email
            count = context.user_data['try']
            count += 1
            context.user_data['try'] = count

            await update.message.reply_text("Inserire nuovamente l'email: ")
            print(context.user_data['try'])

            return USERNAME

        else:
            # Se il numero di tentativi è superato, cancella i dati utente e termina la conversazione
            context.user_data.clear()
            await update.message.reply_text('Numero di tentativi superato premere login per riprovare',
                                            reply_markup=login_markup)
            return ConversationHandler.END


# Funzione per gestire l'annullamento della conversazione di login
async def handle_cancel(update: Update, context: ContextTypes.DEFAULT_TYPE):
    login_key = [
        [KeyboardButton("/login")],
    ]
    login_markup = ReplyKeyboardMarkup(login_key, resize_keyboard=True)

    await update.message.reply_text("Login interrotto!!", reply_markup=login_markup)
    return ConversationHandler.END


# Funzione per gestire il comando di logout
async def handle_logout(update: Update, context: ContextTypes.DEFAULT_TYPE):
    login_key = [
        [KeyboardButton("/login")],
    ]
    login_markup = ReplyKeyboardMarkup(login_key, resize_keyboard=True)

    # Verifica se l'utente è loggato
    if 'logged' in context.user_data and context.user_data['logged']:
        # Esegue il comando di logout tramite il notification server
        response = context.user_data['connect'].logout()

        # Gestione delle risposte del server al comando di logout
        if response == "logout effettuato":
            await update.message.reply_text("Logout effettuato!!"
                                            "Se vuoi effettuare l'accesso premi login.", reply_markup=login_markup)
            context.user_data.clear()
        elif response == "Errore di connessione":
            await update.message.reply_text("Server di autenticazione momentaneamente offline.\n"
                                            "Riprovare piu tardi.", reply_markup=login_markup)
            return ConversationHandler.END
        elif response == "Timeout della richiesta":
            await update.message.reply_text("Timeout della richiesta.\n"
                                            "Riprovare piu tardi.", reply_markup=login_markup)
            return ConversationHandler.END
        elif response == "Errore durante la richiesta":
            await update.message.reply_text("Errore durante la richiesta per favore riprovare.\n"
                                            , reply_markup=login_markup)
            return ConversationHandler.END

    else:
        # Se l'utente non è loggato, invia un messaggio appropriato
        await update.message.reply_text("Devi prima aver effettuato l'accesso per poter effettuare il logout.",
                                        reply_markup=login_markup)

# Configurazione del bot
if __name__ == '__main__':
    with open("config.json", 'r') as file:
        file_content = file.read()

    data = json.loads(file_content)

    # Inizializzazione del bot con il token fornito
    application = ApplicationBuilder().token(data['api_bot']).build()

    # Configurazione del gestore della conversazione di login
    login_handler = ConversationHandler(
        # Definisce quando inizia la conversazione
        entry_points=[CommandHandler("login", login)],
        # Definisce gli "stati" della conversazione e le funzioni associate a ciascuno stato
        states={
            # Quando l'utente è nello stato USERNAME, il bot attende un messaggio di testo
            # e chiama la funzione username per gestire la risposta o la fallbacks qualora si digitasse /cancel.
            USERNAME: [MessageHandler(filters.TEXT & ~ filters.COMMAND, username)],
            # Quando l'utente è nello stato PASSWORD, il bot attende un messaggio di testo
            # e chiama la funzione password per gestire la risposta o la fallbacks qualora si digitasse /cancel.
            PASSWORD: [MessageHandler(filters.TEXT & ~  filters.COMMAND, password)],
        },
        # Definisce cosa succede se l'utente invia il comando "/cancel" durante qualsiasi fase della conversazione.
        # In questo caso, viene eseguito il gestore del comando handle_cancel.
        fallbacks=[CommandHandler("cancel", handle_cancel)],
    )

    # Configurazione dei gestori dei comandi e aggiunta al bot
    start_handler = CommandHandler('start', start)
    logout_handler = CommandHandler('logout', handle_logout)

    application.add_handler(start_handler)
    application.add_handler(login_handler)
    application.add_handler(logout_handler)

    # Avvio del bot
    application.run_polling()
