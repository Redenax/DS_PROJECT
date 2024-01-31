package strutture

type Registrazione interface {
	GetNome() string
	GetCognome() string
	GetActive() string
	Login
}

type Login interface {
	GetEmail() string
	GetPassword() string
	GetIdTg() string
}
