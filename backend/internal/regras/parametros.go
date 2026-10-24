package regras

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/shopspring/decimal"
)

func (p Parametros) cru(nome string) (json.RawMessage, error) {
	raw, ok := p[nome]
	if !ok || len(bytes.TrimSpace(raw)) == 0 || string(bytes.TrimSpace(raw)) == "null" {
		return nil, fmt.Errorf("parâmetro %s ausente", nome)
	}
	return raw, nil
}

// Inteiro lê um parâmetro numérico inteiro (180 ou 180.0; 180.5 é erro).
func (p Parametros) Inteiro(nome string) (int, error) {
	d, err := p.Decimal(nome)
	if err != nil {
		return 0, err
	}
	if !d.Equal(d.Truncate(0)) {
		return 0, fmt.Errorf("parâmetro %s deve ser inteiro", nome)
	}
	return int(d.IntPart()), nil
}

// Decimal lê um parâmetro numérico sem passar por float.
func (p Parametros) Decimal(nome string) (decimal.Decimal, error) {
	raw, err := p.cru(nome)
	if err != nil {
		return decimal.Decimal{}, err
	}
	s := string(bytes.TrimSpace(raw))
	if _, err := strconv.ParseFloat(s, 64); err != nil {
		return decimal.Decimal{}, fmt.Errorf("parâmetro %s deve ser número", nome)
	}
	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("parâmetro %s deve ser número", nome)
	}
	return d, nil
}

// ListaTexto lê uma lista de textos.
func (p Parametros) ListaTexto(nome string) ([]string, error) {
	raw, err := p.cru(nome)
	if err != nil {
		return nil, err
	}
	var l []string
	if err := json.Unmarshal(raw, &l); err != nil {
		return nil, fmt.Errorf("parâmetro %s deve ser lista de textos", nome)
	}
	return l, nil
}

// Texto lê um texto.
func (p Parametros) Texto(nome string) (string, error) {
	raw, err := p.cru(nome)
	if err != nil {
		return "", err
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", fmt.Errorf("parâmetro %s deve ser texto", nome)
	}
	return s, nil
}
