package message

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestTextoPrazoMaximo(t *testing.T) {
	x := Recusa("PRAZO_MAXIMO", map[string]any{"dias": 240, "maximo": 180, "antecipavel_em": "2026-12-21"})
	if x.Titulo != "Prazo acima do limite" {
		t.Errorf("titulo = %q", x.Titulo)
	}
	if x.Mensagem != "Esta nota vence em 240 dias e o fundo compra notas com até 180 dias." {
		t.Errorf("mensagem = %q", x.Mensagem)
	}
	if x.Acao != "Você pode antecipá-la a partir de 21/12/2026." {
		t.Errorf("acao = %q", x.Acao)
	}
}

func TestTextoDadosDeJSON(t *testing.T) {
	original := map[string]any{"tipo": "Cheque", "aceitos": []string{"Duplicata", "Duplicata de Serviço"}}
	b, _ := json.Marshal(original)
	var voltou map[string]any
	_ = json.Unmarshal(b, &voltou)

	a, d := Recusa("TIPO_RECEBIVEL", original), Recusa("TIPO_RECEBIVEL", voltou)
	if a != d {
		t.Fatalf("textos diferentes: %+v vs %+v", a, d)
	}
	if a.Mensagem != "O fundo compra só Duplicata e Duplicata de Serviço, e esta nota é Cheque." {
		t.Fatalf("mensagem = %q", a.Mensagem)
	}
}

func TestTextoValorMinimo(t *testing.T) {
	x := Recusa("VALOR_MINIMO", map[string]any{"valor_centavos": float64(9999), "minimo_centavos": float64(10000)})
	if x.Mensagem != "Esta nota vale R$ 99,99 e o fundo compra notas a partir de R$ 100,00." {
		t.Fatalf("mensagem = %q", x.Mensagem)
	}
}

func TestTextoConcentracao(t *testing.T) {
	x := Recusa("CONCENTRACAO_MAXIMA_SACADO", map[string]any{"percentual_resultante": 12.61, "limite": 10.0, "grupo": "58206002", "grupo_nome": "SERRALHERIA VIOLETA S.A."})
	if x.Mensagem != "Com as notas deste lote, SERRALHERIA VIOLETA S.A. ficaria com 12,61% da carteira do fundo, e o limite é 10%." {
		t.Fatalf("mensagem = %q", x.Mensagem)
	}
}

func TestTextoDuplicidadeValorDiferente(t *testing.T) {
	dados := map[string]any{"vendida_em": "2026-04-27", "numero": "725714", "vencimento": "2026-10-07", "valor_original_centavos": 227808, "valor_centavos": 300000}
	x := Recusa("DUPLICIDADE", dados)
	esperado := "Esta nota (nº 725714, vencimento 07/10/2026) já foi cedida a um fundo em 27/04/2026. O valor registrado na venda foi R$ 2.278,08."
	if x.Mensagem != esperado {
		t.Fatalf("mensagem = %q", x.Mensagem)
	}
	dados["valor_centavos"] = 227808
	if strings.Contains(Recusa("DUPLICIDADE", dados).Mensagem, "valor registrado") {
		t.Fatal("valor igual não deveria citar o valor registrado")
	}
}

func TestTextoFormatoInvalido(t *testing.T) {
	x := Recusa("FORMATO_INVALIDO", map[string]any{"campo": "DATA_VENCIMENTO_ORIGINAL", "valor": "31/02/2026", "problema": "não é uma data válida (use dd/mm/aaaa)", "linha": 1204})
	if x.Mensagem != `Na coluna "vencimento" (DATA_VENCIMENTO_ORIGINAL), o valor "31/02/2026" não é uma data válida (use dd/mm/aaaa).` {
		t.Fatalf("mensagem = %q", x.Mensagem)
	}
}

var exemplos = map[string]map[string]any{
	"CNPJ_SACADO_VALIDO":          {"documento": "77.788.899/0001-80"},
	"SITUACAO_CADASTRAL_SACADO":   {"situacao": "INAPTA", "aceitas": []string{"ATIVA"}},
	"TIPO_RECEBIVEL":              {"tipo": "Cheque", "aceitos": []string{"Duplicata"}},
	"VALOR_MINIMO":                {"valor_centavos": 5000, "minimo_centavos": 10000},
	"NAO_VENCIDA":                 {"vencimento": "2026-10-01", "data_base": "2026-10-22"},
	"PRAZO_MAXIMO":                {"dias": 240, "maximo": 180, "antecipavel_em": "2026-12-21"},
	"PRAZO_MINIMO":                {"dias": 10, "minimo": 30},
	"CONCENTRACAO_MAXIMA_SACADO":  {"percentual_resultante": 12.6, "limite": 10.0, "grupo": "58206002", "grupo_nome": "X"},
	"CONCENTRACAO_MAXIMA_CEDENTE": {"percentual_resultante": 9.1, "limite": 8.0, "grupo": "10062050000157", "grupo_nome": "Y"},
	"DUPLICIDADE":                 {"vendida_em": "2026-04-27", "numero": "1", "vencimento": "2026-11-01", "valor_original_centavos": 100, "valor_centavos": 100},
	"DUPLICADA_NO_ARQUIVO":        {"linha_original": 2},
	"FORMATO_INVALIDO":            {"campo": "VALOR_NOMINAL", "valor": "abc", "problema": "não é um valor válido (use 1234,56)", "linha": 3},
}

func TestTodosCodigosTemTexto(t *testing.T) {
	if len(exemplos) != 12 {
		t.Fatalf("esperava 12 códigos, há %d", len(exemplos))
	}
	for codigo, dados := range exemplos {
		x := Recusa(codigo, dados)
		if Titulo(codigo) == "" || x.Titulo != Titulo(codigo) {
			t.Errorf("%s sem título", codigo)
		}
		if x.Mensagem == "" || strings.Contains(x.Mensagem, "{") || strings.Contains(x.Mensagem, "%!") {
			t.Errorf("%s mensagem ruim: %q", codigo, x.Mensagem)
		}
		if x.Acao == "" || AcaoResumo(codigo) == "" || strings.Contains(AcaoResumo(codigo), "{") {
			t.Errorf("%s sem ação", codigo)
		}
	}
	if AcaoResumo("PRAZO_MAXIMO") != "Antecipe essas notas mais perto do vencimento." {
		t.Errorf("AcaoResumo(PRAZO_MAXIMO) = %q", AcaoResumo("PRAZO_MAXIMO"))
	}
}

func TestCodigoDesconhecido(t *testing.T) {
	x := Recusa("REGRA_X", nil)
	if x.Titulo == "" || x.Mensagem == "" {
		t.Fatalf("texto genérico ausente: %+v", x)
	}
}

func TestTextoCampoVazio(t *testing.T) {
	x := Recusa("FORMATO_INVALIDO", map[string]any{"campo": "VALOR_NOMINAL", "valor": "", "problema": "está vazio", "linha": 3})
	if x.Mensagem != `A coluna "valor" (VALOR_NOMINAL) está vazia.` {
		t.Fatalf("mensagem = %q", x.Mensagem)
	}
}

func TestTextoLimiteComDecimais(t *testing.T) {
	x := Recusa("CONCENTRACAO_MAXIMA_CEDENTE", map[string]any{"percentual_resultante": 10.04, "limite": 7.5, "grupo": "x", "grupo_nome": "Y"})
	if x.Mensagem != "Com as notas deste lote, a empresa ficaria com 10,04% da carteira do fundo, e o limite é 7,5%." {
		t.Fatalf("mensagem = %q", x.Mensagem)
	}
}
