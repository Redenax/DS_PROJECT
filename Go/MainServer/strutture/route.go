package strutture

import "log"

type Routes struct {
	nome         string
	partenza     string
	destinazione string
	email        string
}

func NewRoutes() *Routes {
	return &Routes{}
}

// Metodo per inizializzare una nuova Route con valori specifici
func (r *Routes) Initialize(nome, partenza, destinazione, email string) {
	r.nome = nome
	r.partenza = partenza
	r.destinazione = destinazione
	r.email = email
}

func (r *Routes) GetNome() string {
	return r.nome
}

func (r *Routes) SetNome(nome string) {
	r.nome = nome
}

func (r *Routes) GetPartenza() string {
	return r.partenza
}

func (r *Routes) SetPartenza(partenza string) {
	r.partenza = partenza
}

func (r *Routes) GetDestinazione() string {
	return r.destinazione
}

func (r *Routes) SetDestinazione(destinazione string) {
	r.destinazione = destinazione
}

func (r *Routes) GetEmail() string {
	return r.email
}

func (r *Routes) SetEmail(email string) {
	r.email = email
}

func (r *Routes) ToString() {
	log.Printf("Nome: %s, Partenza: %s, Destinazione: %s, Email: %s",
		r.nome, r.partenza, r.destinazione, r.email)
}
