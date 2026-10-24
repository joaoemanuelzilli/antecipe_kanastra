// Package textos transforma (código, dados) de uma recusa no texto que a empresa entende,
// sempre na ordem decisão → regra → consequência → ação. Tela e API usam o mesmo texto.
package message

import (
	"encoding/json"
	"strconv"

	"antecipe/internal/platform/format"
)

// Texto de uma recusa.
type Texto struct {
	Titulo   string `json:"titulo"`
	Mensagem string `json:"mensagem"`
	Acao     string `json:"acao"`
}

var titulos = map[string]string{
	"CNPJ_SACADO_VALIDO":          "CNPJ do cliente inválido",
	"SITUACAO_CADASTRAL_SACADO":   "CNPJ do cliente com pendência",
	"TIPO_RECEBIVEL":              "Tipo de nota não aceito",
	"VALOR_MINIMO":                "Valor abaixo do mínimo",
	"NAO_VENCIDA":                 "Nota vencida",
	"PRAZO_MAXIMO":                "Prazo acima do limite",
	"PRAZO_MINIMO":                "Prazo abaixo do mínimo",
	"CONCENTRACAO_MAXIMA_SACADO":  "Limite por cliente atingido",
	"CONCENTRACAO_MAXIMA_CEDENTE": "Limite por empresa atingido",
	"DUPLICIDADE":                 "Nota já vendida",
	"DUPLICADA_NO_ARQUIVO":        "Nota repetida no arquivo",
	"FORMATO_INVALIDO":            "Linha com erro",
}

var acoes = map[string]string{
	"CNPJ_SACADO_VALIDO":          "Confira o CNPJ com seu cliente e envie a nota de novo.",
	"SITUACAO_CADASTRAL_SACADO":   "Confirme os dados com seu cliente antes de antecipar esta nota.",
	"TIPO_RECEBIVEL":              "Esta nota não pode ser antecipada neste fundo.",
	"VALOR_MINIMO":                "Notas de valor menor não podem ser antecipadas aqui.",
	"NAO_VENCIDA":                 "Cobre esta nota diretamente do seu cliente.",
	"PRAZO_MINIMO":                "Esta nota está perto demais do vencimento para ser antecipada.",
	"CONCENTRACAO_MAXIMA_SACADO":  "Tente com notas de outros clientes ou aguarde o vencimento das notas atuais desse cliente.",
	"CONCENTRACAO_MAXIMA_CEDENTE": "Aguarde o vencimento de parte das notas já vendidas antes de antecipar mais.",
	"DUPLICIDADE":                 "Não é preciso enviar de novo.",
	"DUPLICADA_NO_ARQUIVO":        "Remova a linha repetida. A primeira ocorrência foi avaliada normalmente.",
	"FORMATO_INVALIDO":            "Corrija a linha e envie o arquivo de novo.",
}

// Titulo devolve o título curto do código.
func Titulo(codigo string) string {
	if t, ok := titulos[codigo]; ok {
		return t
	}
	return "Regra não cumprida"
}

// AcaoResumo devolve a ação sem dados específicos de uma nota (para o resumo dos motivos).
func AcaoResumo(codigo string) string {
	if codigo == "PRAZO_MAXIMO" {
		return "Antecipe essas notas mais perto do vencimento."
	}
	if a, ok := acoes[codigo]; ok {
		return a
	}
	return "Fale com o fundo para entender esta recusa."
}

// Recusa monta o texto. Aceita dados vindos do Go (int, []string) ou do banco (float64, []any).
func Recusa(codigo string, dados map[string]any) Texto {
	t := Texto{Titulo: Titulo(codigo), Acao: AcaoResumo(codigo)}
	d := leitor(dados)
	switch codigo {
	case "CNPJ_SACADO_VALIDO":
		t.Mensagem = "O CNPJ de quem paga a nota (" + d.texto("documento") + ") não é válido: os dígitos verificadores não conferem."
	case "SITUACAO_CADASTRAL_SACADO":
		t.Mensagem = "O CNPJ de quem paga a nota está " + d.texto("situacao") + " na Receita, e o fundo só compra de clientes com situação " + format.Lista(d.lista("aceitas")) + "."
	case "TIPO_RECEBIVEL":
		t.Mensagem = "O fundo compra só " + format.Lista(d.lista("aceitos")) + ", e esta nota é " + d.texto("tipo") + "."
	case "VALOR_MINIMO":
		t.Mensagem = "Esta nota vale " + format.Reais(d.inteiro("valor_centavos")) + " e o fundo compra notas a partir de " + format.Reais(d.inteiro("minimo_centavos")) + "."
	case "NAO_VENCIDA":
		t.Mensagem = "Esta nota venceu em " + format.DataISO(d.texto("vencimento")) + " (data-base " + format.DataISO(d.texto("data_base")) + ") e o fundo só compra notas a vencer."
	case "PRAZO_MAXIMO":
		t.Mensagem = "Esta nota vence em " + format.Inteiro(int(d.inteiro("dias"))) + " dias e o fundo compra notas com até " + format.Inteiro(int(d.inteiro("maximo"))) + " dias."
		t.Acao = "Você pode antecipá-la a partir de " + format.DataISO(d.texto("antecipavel_em")) + "."
	case "PRAZO_MINIMO":
		t.Mensagem = "Esta nota vence em " + format.Inteiro(int(d.inteiro("dias"))) + " dias e o fundo compra notas com pelo menos " + format.Inteiro(int(d.inteiro("minimo"))) + " dias."
	case "CONCENTRACAO_MAXIMA_SACADO":
		t.Mensagem = "Com esta nota, " + d.texto("grupo_nome") + " ficaria com " + format.Percentual(d.real("percentual_resultante"), 1) + " da carteira do fundo, e o limite é " + format.Percentual(d.real("limite"), 0) + "."
	case "CONCENTRACAO_MAXIMA_CEDENTE":
		t.Mensagem = "Com esta nota, sua empresa ficaria com " + format.Percentual(d.real("percentual_resultante"), 1) + " da carteira do fundo, e o limite é " + format.Percentual(d.real("limite"), 0) + "."
	case "DUPLICIDADE":
		t.Mensagem = "Esta nota (nº " + d.texto("numero") + ", vencimento " + format.DataISO(d.texto("vencimento")) + ") já foi cedida a um fundo em " + format.DataISO(d.texto("vendida_em")) + "."
		if orig, atual := d.inteiro("valor_original_centavos"), d.inteiro("valor_centavos"); orig > 0 && orig != atual {
			t.Mensagem += " O valor registrado na venda foi " + format.Reais(orig) + "."
		}
	case "DUPLICADA_NO_ARQUIVO":
		t.Mensagem = "Esta nota já aparece na linha " + format.Inteiro(int(d.inteiro("linha_original"))) + " deste arquivo."
	case "FORMATO_INVALIDO":
		if d.texto("campo") == "linha" {
			t.Mensagem = "Esta linha " + d.texto("problema") + "."
		} else {
			t.Mensagem = "Na coluna " + d.texto("campo") + ", o valor \"" + d.texto("valor") + "\" " + d.texto("problema") + "."
		}
	default:
		t.Mensagem = "Esta nota não cumpriu a regra " + codigo + " do fundo."
	}
	return t
}

type leitor map[string]any

func (d leitor) texto(k string) string {
	switch v := d[k].(type) {
	case string:
		return v
	case nil:
		return ""
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

func (d leitor) inteiro(k string) int64 {
	switch v := d[k].(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case json.Number:
		n, _ := v.Int64()
		return n
	case string:
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	}
	return 0
}

func (d leitor) real(k string) float64 {
	switch v := d[k].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	}
	return 0
}

func (d leitor) lista(k string) []string {
	switch v := d[k].(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, x := range v {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}
