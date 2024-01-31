package strutture

import "log"

type Authentication struct {
	email    string
	password string
	idTg     string
}

func NewAuthentication() *Authentication {
	return &Authentication{}
}

func (a *Authentication) Initialize(email, password, idTg string) {
	a.email = email
	a.password = password
	a.idTg = idTg
}

// Implementa i metodi dell'interfaccia Login per Authentication
func (a *Authentication) GetEmail() string {
	return a.email
}

func (a *Authentication) SetEmail(email string) {
	a.email = email
}

func (a *Authentication) GetPassword() string {
	return a.password
}

func (a *Authentication) SetPassword(password string) {
	a.password = password
}

func (a *Authentication) GetIdTg() string {
	return a.idTg
}

func (a *Authentication) SetIdTg(idTg string) {
	a.idTg = idTg
}

func (a *Authentication) ToString() {
	log.Printf("Email: %s, Password: %s, IdTg: %s",
		a.email, a.password, a.idTg)
}
