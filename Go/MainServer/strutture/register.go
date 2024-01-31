package strutture

import "log"

type Utenti struct {
	nome     string
	cognome  string
	email    string
	password string
	idTg     string
	active   string
}

func NewUtenti() *Utenti {
	return &Utenti{}
}

func (u *Utenti) Initialize(nome, cognome, email, password, idTg, active string) {
	u.nome = nome
	u.cognome = cognome
	u.email = email
	u.password = password
	u.idTg = idTg
	u.active = active
}

// Implementa i metodi dell'interfaccia Registrazione per Utenti
func (u *Utenti) GetNome() string {
	return u.nome
}

func (u *Utenti) SetNome(nome string) {
	u.nome = nome
}

func (u *Utenti) GetCognome() string {
	return u.cognome
}

func (u *Utenti) SetCognome(cognome string) {
	u.cognome = cognome
}

func (u *Utenti) GetEmail() string {
	return u.email
}

func (u *Utenti) SetEmail(email string) {
	u.email = email
}

func (u *Utenti) GetPassword() string {
	return u.password
}

func (u *Utenti) SetPassword(password string) {
	u.password = password
}

func (u *Utenti) GetIdTg() string {
	return u.idTg
}

func (u *Utenti) SetIdTg(idTg string) {
	u.idTg = idTg
}

func (u *Utenti) GetActive() string {
	return u.active
}

func (u *Utenti) SetActive(active string) {
	u.active = active
}

func (u *Utenti) ToString() {
	log.Printf("Nome: %s, Cognome: %s, Email: %s, Password: %s, IdTg: %s, Active: %s",
		u.nome, u.cognome, u.email, u.password, u.idTg, u.active)
}
