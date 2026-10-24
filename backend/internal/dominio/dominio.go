// Package dominio tem os tipos compartilhados: a nota lida do CSV e o resultado da avaliação.
package dominio

import "time"

// Nota é uma linha do CSV já interpretada.
type Nota struct {
	Linha int // linha física no arquivo: cabeçalho = 1, primeira nota = 2

	NomeFundo, DocFundo     string // como no arquivo (com TrimSpace)
	NomeCedente, DocCedente string
	NomeSacado, DocSacado   string
	Numero, Tipo            string

	ValorCentavos int64
	Vencimento    time.Time // UTC 00:00; zero se inválida
	Aquisicao     time.Time // zero se vazia

	ErroFormato *ErroCampo // preenchido quando a linha não pôde ser lida
}

// ErroCampo descreve o primeiro problema de leitura de uma linha.
type ErroCampo struct {
	Campo    string `json:"campo"`
	Valor    string `json:"valor"`
	Problema string `json:"problema"`
}

// Status da nota depois da avaliação.
type Status uint8

const (
	Aprovada Status = 1
	Recusada Status = 2
)

func (s Status) String() string {
	if s == Aprovada {
		return "aprovada"
	}
	return "recusada"
}

// Recusa é uma regra que a nota não cumpriu, com os dados para explicar o motivo.
type Recusa struct {
	Codigo string         `json:"codigo"`
	Dados  map[string]any `json:"dados"`
}

// Resultado é a avaliação de uma nota. Recusas[0] é o motivo principal.
type Resultado struct {
	Nota             *Nota
	Chave            string
	Dias             int
	Status           Status
	DescontoCentavos int64
	LiquidoCentavos  int64
	Recusas          []Recusa
}
