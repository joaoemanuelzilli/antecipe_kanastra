// Package precificacao calcula o preço de compra de cada nota.
//
// Métodos: DESCONTO_COMPOSTO (padrão, desconto racional composto: líquido = face ÷ (1+i)^(dias/base))
// e DESCONTO_SIMPLES (desconto = face × i × dias/base). Valores em centavos, arredondados por nota.
package precificacao

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/shopspring/decimal"

	"antecipe/internal/formato"
)

const (
	Composto = "DESCONTO_COMPOSTO"
	Simples  = "DESCONTO_SIMPLES"
)

func init() { decimal.MarshalJSONWithoutQuotes = true }

// Config é o que define o preço: método, taxa mensal (%) e base em dias.
type Config struct {
	Metodo               string          `json:"metodo"`
	TaxaMensalPercentual decimal.Decimal `json:"taxa_mensal_percentual"`
	BaseDias             int             `json:"base_dias"`
}

// Metodos lista os métodos aceitos.
func Metodos() []string { return []string{Composto, Simples} }

// Descricao devolve o texto curto para a tela: "Juros compostos · 2% ao mês".
func (c Config) Descricao() string {
	nome := "Juros compostos"
	if c.Metodo == Simples {
		nome = "Juros simples"
	}
	return nome + " · " + pct(c.TaxaMensalPercentual) + " ao mês"
}

// Efetiva aplica as sobrescritas do YAML sobre a precificação das regras publicadas.
func Efetiva(base Config, metodo *string, taxa *decimal.Decimal, baseDias *int) (Config, []string) {
	c := base
	var sobrescritos []string
	if metodo != nil {
		c.Metodo = *metodo
		sobrescritos = append(sobrescritos, "metodo")
	}
	if taxa != nil {
		c.TaxaMensalPercentual = *taxa
		sobrescritos = append(sobrescritos, "taxa_mensal_percentual")
	}
	if baseDias != nil {
		c.BaseDias = *baseDias
		sobrescritos = append(sobrescritos, "base_dias")
	}
	return c, sobrescritos
}

// Calculadora aplica uma Config. É segura para uso concorrente.
type Calculadora struct {
	cfg  Config
	i    decimal.Decimal // taxa ÷ 100
	base decimal.Decimal

	mu      sync.RWMutex
	fatores map[int]decimal.Decimal // (1+i)^(dias/base), memorizado por prazo
}

// Nova valida a Config e cria a calculadora.
func Nova(c Config) (*Calculadora, error) {
	if c.Metodo != Composto && c.Metodo != Simples {
		return nil, fmt.Errorf("precificação: método desconhecido %q", c.Metodo)
	}
	if c.BaseDias < 1 {
		return nil, fmt.Errorf("precificação: base_dias deve ser maior que zero")
	}
	if c.TaxaMensalPercentual.IsNegative() {
		return nil, fmt.Errorf("precificação: taxa negativa")
	}
	return &Calculadora{
		cfg:     c,
		i:       c.TaxaMensalPercentual.Div(decimal.NewFromInt(100)),
		base:    decimal.NewFromInt(int64(c.BaseDias)),
		fatores: map[int]decimal.Decimal{},
	}, nil
}

// Config devolve a configuração aplicada.
func (c *Calculadora) Config() Config { return c.cfg }

// Calcular devolve desconto e líquido em centavos. Prazo ≤ 0 não tem desconto.
func (c *Calculadora) Calcular(faceCentavos int64, dias int) (descontoCentavos, liquidoCentavos int64) {
	if dias <= 0 {
		return 0, faceCentavos
	}
	face := decimal.NewFromInt(faceCentavos)
	if c.cfg.Metodo == Simples {
		d := face.Mul(c.i).Mul(decimal.NewFromInt(int64(dias))).DivRound(c.base, 0).IntPart()
		return d, faceCentavos - d
	}
	l := face.DivRound(c.fator(dias), 0).IntPart()
	return faceCentavos - l, l
}

func (c *Calculadora) fator(dias int) decimal.Decimal {
	c.mu.RLock()
	f, ok := c.fatores[dias]
	c.mu.RUnlock()
	if ok {
		return f
	}
	exp := decimal.NewFromInt(int64(dias)).DivRound(c.base, 20)
	f, err := decimal.NewFromInt(1).Add(c.i).PowWithPrecision(exp, 20)
	if err != nil {
		// (1+i) é sempre positivo; erro aqui seria bug de precisão. Sem desconto é o lado seguro.
		f = decimal.NewFromInt(1)
	}
	c.mu.Lock()
	c.fatores[dias] = f
	c.mu.Unlock()
	return f
}

// Texto explica a conta de uma nota aprovada.
func (c *Calculadora) Texto(faceCentavos int64, dias int) string {
	desconto, liquido := c.Calcular(faceCentavos, dias)
	periodo := "(" + strconv.Itoa(dias) + " ÷ " + strconv.Itoa(c.cfg.BaseDias) + ")"
	taxa := pct(c.cfg.TaxaMensalPercentual)
	if c.cfg.Metodo == Simples {
		return formato.Reais(faceCentavos) + " × " + taxa + " × " + periodo + " = " + formato.Reais(desconto) +
			" de desconto → você recebe " + formato.Reais(liquido)
	}
	return formato.Reais(faceCentavos) + " ÷ (1 + " + taxa + ")^" + periodo + " = " + formato.Reais(liquido) +
		" → desconto de " + formato.Reais(desconto)
}

func pct(d decimal.Decimal) string {
	return strings.Replace(d.String(), ".", ",", 1) + "%"
}
