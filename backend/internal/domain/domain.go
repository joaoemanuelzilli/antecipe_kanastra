// Package dominio tem os tipos compartilhados: a nota lida do CSV e o resultado da avaliação.
package domain

import (
	"strings"
	"time"

	"antecipe/internal/platform/document"
)

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

// Vendida é uma nota que já está no estoque (de qualquer fundo) ou numa cessão.
type Vendida struct {
	Em            time.Time
	ValorCentavos int64
}

// ItemEstoque é uma nota da carteira (estoque), com o que a concentração precisa.
type ItemEstoque struct {
	FundoDoc, SacadoDoc, CedenteDoc string // normalizados
	ValorCentavos                   int64
}

// ContextoEstoque é o que o motor precisa saber do estoque:
// Vendidas = notas vendidas pelo portal (cessões); Estoques = carteira de cada fundo; Itens = notas da carteira por chave.
type ContextoEstoque struct {
	Vendidas map[string]Vendida
	Estoques map[string]EstoqueFundo // chave: CNPJ normalizado do fundo
	Itens    map[string]ItemEstoque
}

// EstoqueFundo soma a carteira de um fundo, por documento normalizado.
type EstoqueFundo struct {
	TotalCentavos int64
	PorSacado     map[string]int64
	PorCedente    map[string]int64
}

// Chave identifica a nota para duplicidade: cedente + número + vencimento (o valor fica de fora de propósito).
// O número é canonizado (maiúsculas, só letras e dígitos, sem zeros à esquerda) para que
// "000351153", "351.153" e "351153" não escapem da detecção. Na dúvida, é melhor recusar como
// duplicata do que vender a mesma nota duas vezes.
func Chave(n *Nota) string {
	return document.Normalizar(n.DocCedente) + "|" + NumeroCanonico(n.Numero) + "|" + n.Vencimento.Format("2006-01-02")
}

// NumeroCanonico normaliza o número da nota para a chave de duplicidade.
func NumeroCanonico(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range strings.ToUpper(s) {
		if (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') {
			if b.Len() == 0 && r == '0' {
				continue
			}
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 && strings.ContainsRune(s, '0') {
		return "0"
	}
	return b.String()
}
