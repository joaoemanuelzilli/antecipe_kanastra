package rules

import (
	"encoding/json"
	"reflect"
	"strings"

	"github.com/shopspring/decimal"

	"antecipe/internal/platform/format"
)

// Resumo descreve em português o que mudou entre duas versões ("PRAZO_MAXIMO: 180 → 120 dias").
func Resumo(antes, depois Regras) []string {
	var out []string
	porCodigo := make(map[string]Validador, len(antes.Validadores))
	for _, v := range antes.Validadores {
		porCodigo[v.Codigo] = v
	}
	vistos := map[string]bool{}
	for _, d := range depois.Validadores {
		vistos[d.Codigo] = true
		a, existia := porCodigo[d.Codigo]
		if !existia {
			out = append(out, d.Codigo+": adicionada")
			continue
		}
		if a.Ativo != d.Ativo {
			if d.Ativo {
				out = append(out, d.Codigo+": ligada")
			} else {
				out = append(out, d.Codigo+": desligada")
			}
		}
		item, _ := ItemPorCodigo(d.Codigo)
		for _, nome := range nomesParametros(a, d, item) {
			if mesmoValor(a.Parametros[nome], d.Parametros[nome]) {
				continue
			}
			pc := parametroDoCatalogo(item, nome)
			prefixo := d.Codigo
			if len(item.Parametros) > 1 {
				prefixo += " (" + pc.Rotulo + ")"
			}
			out = append(out, prefixo+": "+valorLegivel(a.Parametros[nome], pc)+" → "+valorLegivel(d.Parametros[nome], pc)+sufixo(pc))
		}
	}
	for _, a := range antes.Validadores {
		if !vistos[a.Codigo] {
			out = append(out, a.Codigo+": removida")
		}
	}

	pa, pd := antes.Precificacao, depois.Precificacao
	if pa.Metodo != pd.Metodo {
		out = append(out, "Método: "+pa.Metodo+" → "+pd.Metodo)
	}
	if !pa.TaxaMensalPercentual.Equal(pd.TaxaMensalPercentual) {
		out = append(out, "Taxa: "+decimalBR(pa.TaxaMensalPercentual)+"% → "+decimalBR(pd.TaxaMensalPercentual)+"% ao mês")
	}
	if pa.BaseDias != pd.BaseDias {
		out = append(out, "Base: "+format.Inteiro(pa.BaseDias)+" → "+format.Inteiro(pd.BaseDias)+" dias")
	}
	return out
}

func nomesParametros(a, d Validador, item ItemCatalogo) []string {
	var nomes []string
	visto := map[string]bool{}
	for _, pc := range item.Parametros {
		nomes = append(nomes, pc.Nome)
		visto[pc.Nome] = true
	}
	for _, p := range []Parametros{a.Parametros, d.Parametros} {
		for k := range p {
			if !visto[k] {
				nomes = append(nomes, k)
				visto[k] = true
			}
		}
	}
	return nomes
}

func parametroDoCatalogo(item ItemCatalogo, nome string) ParametroCatalogo {
	for _, pc := range item.Parametros {
		if pc.Nome == nome {
			return pc
		}
	}
	return ParametroCatalogo{Nome: nome, Rotulo: nome}
}

func mesmoValor(a, b json.RawMessage) bool {
	var va, vb any
	_ = json.Unmarshal(a, &va)
	_ = json.Unmarshal(b, &vb)
	return reflect.DeepEqual(va, vb)
}

func valorLegivel(raw json.RawMessage, pc ParametroCatalogo) string {
	if len(raw) == 0 {
		return "(vazio)"
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return string(raw)
	}
	switch x := v.(type) {
	case []any:
		itens := make([]string, 0, len(x))
		for _, it := range x {
			if s, ok := it.(string); ok {
				itens = append(itens, s)
			}
		}
		return format.Lista(itens)
	case float64:
		d, err := decimal.NewFromString(strings.TrimSpace(string(raw)))
		if err != nil {
			return string(raw)
		}
		switch pc.Unidade {
		case "R$":
			return format.Reais(d.Mul(decimal.NewFromInt(100)).Round(0).IntPart())
		case "%":
			return decimalBR(d) + "%"
		}
		return decimalBR(d)
	case string:
		return x
	}
	return string(raw)
}

func sufixo(pc ParametroCatalogo) string {
	if pc.Unidade == "dias" {
		return " dias"
	}
	return ""
}

func decimalBR(d decimal.Decimal) string {
	return strings.Replace(d.String(), ".", ",", 1)
}
