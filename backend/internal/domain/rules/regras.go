// Package regras representa a configuração do fundo (validadores_antecipe.json e suas versões).
// Nenhum valor de regra mora no código: tudo vem daqui.
package rules

import (
	"encoding/json"
	"fmt"

	"github.com/shopspring/decimal"
)

func init() {
	// taxa_mensal_percentual sai como número no JSON (2, não "2"), igual ao arquivo original.
	decimal.MarshalJSONWithoutQuotes = true
}

// Regras é o conteúdo do validadores_antecipe.json.
type Regras struct {
	Fundo        Fundo        `json:"fundo"`
	Validadores  []Validador  `json:"validadores"`
	Precificacao Precificacao `json:"precificacao"`
}

type Fundo struct {
	Nome string `json:"nome"`
	CNPJ string `json:"cnpj"`
}

type Validador struct {
	Codigo     string     `json:"codigo"`
	Ativo      bool       `json:"ativo"`
	Parametros Parametros `json:"parametros"`
}

// Parametros guarda os valores crus; os acessores tipados validam na leitura.
type Parametros map[string]json.RawMessage

type Precificacao struct {
	Metodo               string          `json:"metodo"`
	TaxaMensalPercentual decimal.Decimal `json:"taxa_mensal_percentual"`
	BaseDias             int             `json:"base_dias"`
}

// Ler interpreta o JSON de regras.
func Ler(data []byte) (Regras, error) {
	var r Regras
	if err := json.Unmarshal(data, &r); err != nil {
		return Regras{}, fmt.Errorf("regras: JSON inválido: %w", err)
	}
	for i := range r.Validadores {
		if r.Validadores[i].Parametros == nil {
			r.Validadores[i].Parametros = Parametros{}
		}
	}
	return r, nil
}

// Clonar devolve uma cópia independente (os mapas de parâmetros não são compartilhados).
func (r Regras) Clonar() Regras {
	c := r
	c.Validadores = make([]Validador, len(r.Validadores))
	for i, v := range r.Validadores {
		p := make(Parametros, len(v.Parametros))
		for k, raw := range v.Parametros {
			p[k] = append(json.RawMessage(nil), raw...)
		}
		v.Parametros = p
		c.Validadores[i] = v
	}
	return c
}
