package validator

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"antecipe/internal/domain"
	"antecipe/internal/domain/rules"
	"antecipe/internal/platform/document"
)

var dataBase = time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)

func ctx(situacao string) Ctx {
	return Ctx{DataBase: dataBase, Situacao: func(string) string { return situacao }}
}

func params(js string) rules.Parametros {
	var p rules.Parametros
	if err := json.Unmarshal([]byte(js), &p); err != nil {
		panic(err)
	}
	return p
}

func item(n domain.Nota) Item {
	dias := int(n.Vencimento.Sub(dataBase).Hours() / 24)
	return Item{Nota: &n, Dias: dias, SacadoNorm: document.Normalizar(n.DocSacado)}
}

func nota(mod func(*domain.Nota)) domain.Nota {
	n := domain.Nota{
		Linha: 2, DocSacado: "22.144.961/0001-02", Tipo: "Duplicata",
		ValorCentavos: 1000000, Vencimento: dataBase.AddDate(0, 0, 30),
	}
	if mod != nil {
		mod(&n)
	}
	return n
}

func rodar(t *testing.T, codigo string, n domain.Nota, p string, c Ctx) (bool, map[string]any) {
	t.Helper()
	f, ok := PorNota[codigo]
	if !ok {
		t.Fatalf("validador %s ausente", codigo)
	}
	okRegra, dados, err := f(item(n), params(p), c)
	if err != nil {
		t.Fatalf("%s: erro inesperado: %v", codigo, err)
	}
	return okRegra, dados
}

func igual(t *testing.T, got, esperado map[string]any) {
	t.Helper()
	g, _ := json.Marshal(got)
	e, _ := json.Marshal(esperado)
	if string(g) != string(e) {
		t.Fatalf("dados = %s, esperado %s", g, e)
	}
}

func TestCNPJ(t *testing.T) {
	ok, dados := rodar(t, "CNPJ_SACADO_VALIDO", nota(func(n *domain.Nota) { n.DocSacado = "77.788.899/0001-80" }), `{}`, ctx("ATIVA"))
	if ok {
		t.Fatal("deveria falhar")
	}
	igual(t, dados, map[string]any{"documento": "77.788.899/0001-80"})
	if ok, _ := rodar(t, "CNPJ_SACADO_VALIDO", nota(func(n *domain.Nota) { n.DocSacado = "12.ABC.345/01DE-35" }), `{}`, ctx("ATIVA")); !ok {
		t.Fatal("alfanumérico deveria passar")
	}
}

func TestSituacao(t *testing.T) {
	ok, dados := rodar(t, "SITUACAO_CADASTRAL_SACADO", nota(nil), `{"situacoes_aceitas":["ATIVA"]}`, ctx("INAPTA"))
	if ok {
		t.Fatal("deveria falhar")
	}
	igual(t, dados, map[string]any{"situacao": "INAPTA", "aceitas": []string{"ATIVA"}})
	if ok, _ := rodar(t, "SITUACAO_CADASTRAL_SACADO", nota(nil), `{"situacoes_aceitas":["ATIVA"]}`, ctx("ATIVA")); !ok {
		t.Fatal("ATIVA deveria passar")
	}
}

func TestTipo(t *testing.T) {
	p := `{"tipos_aceitos":["Duplicata","Duplicata de Serviço"]}`
	if ok, _ := rodar(t, "TIPO_RECEBIVEL", nota(func(n *domain.Nota) { n.Tipo = " duplicata de servico " }), p, ctx("ATIVA")); !ok {
		t.Fatal("sem acento e minúsculo deveria passar")
	}
	ok, dados := rodar(t, "TIPO_RECEBIVEL", nota(func(n *domain.Nota) { n.Tipo = "Cheque" }), p, ctx("ATIVA"))
	if ok {
		t.Fatal("Cheque deveria falhar")
	}
	igual(t, dados, map[string]any{"tipo": "Cheque", "aceitos": []string{"Duplicata", "Duplicata de Serviço"}})
}

func TestValorMinimo(t *testing.T) {
	p := `{"valor_minimo":100.0}`
	if ok, _ := rodar(t, "VALOR_MINIMO", nota(func(n *domain.Nota) { n.ValorCentavos = 10000 }), p, ctx("ATIVA")); !ok {
		t.Fatal("exatamente o mínimo deveria passar")
	}
	ok, dados := rodar(t, "VALOR_MINIMO", nota(func(n *domain.Nota) { n.ValorCentavos = 9999 }), p, ctx("ATIVA"))
	if ok {
		t.Fatal("abaixo do mínimo deveria falhar")
	}
	igual(t, dados, map[string]any{"valor_centavos": 9999, "minimo_centavos": 10000})
}

func TestNaoVencida(t *testing.T) {
	ok, dados := rodar(t, "NAO_VENCIDA", nota(func(n *domain.Nota) { n.Vencimento = dataBase }), `{}`, ctx("ATIVA"))
	if ok {
		t.Fatal("vencimento = data-base deveria falhar")
	}
	igual(t, dados, map[string]any{"vencimento": "2026-10-22", "data_base": "2026-10-22"})
	if ok, _ := rodar(t, "NAO_VENCIDA", nota(func(n *domain.Nota) { n.Vencimento = dataBase.AddDate(0, 0, 1) }), `{}`, ctx("ATIVA")); !ok {
		t.Fatal("vence amanhã deveria passar")
	}
}

func TestPrazoMaximo(t *testing.T) {
	p := `{"prazo_maximo_dias":180}`
	if ok, _ := rodar(t, "PRAZO_MAXIMO", nota(func(n *domain.Nota) { n.Vencimento = dataBase.AddDate(0, 0, 180) }), p, ctx("ATIVA")); !ok {
		t.Fatal("exatamente 180 deveria passar")
	}
	ok, dados := rodar(t, "PRAZO_MAXIMO", nota(func(n *domain.Nota) { n.Vencimento = time.Date(2027, 6, 19, 0, 0, 0, 0, time.UTC) }), p, ctx("ATIVA"))
	if ok {
		t.Fatal("240 dias deveria falhar")
	}
	igual(t, dados, map[string]any{"dias": 240, "maximo": 180, "antecipavel_em": "2026-12-21"})
}

func TestPrazoMinimo(t *testing.T) {
	ok, dados := rodar(t, "PRAZO_MINIMO", nota(func(n *domain.Nota) { n.Vencimento = dataBase.AddDate(0, 0, 29) }), `{"prazo_minimo_dias":30}`, ctx("ATIVA"))
	if ok {
		t.Fatal("29 dias com mínimo 30 deveria falhar")
	}
	igual(t, dados, map[string]any{"dias": 29, "minimo": 30})
}

func TestParametroInvalido(t *testing.T) {
	_, _, err := PorNota["PRAZO_MAXIMO"](item(nota(nil)), params(`{"prazo_maximo_dias":"abc"}`), ctx("ATIVA"))
	if err == nil {
		t.Fatal("parâmetro inválido deveria dar erro")
	}
}

func TestConhecido(t *testing.T) {
	for _, c := range []string{"CNPJ_SACADO_VALIDO", "PRAZO_MINIMO", "CONCENTRACAO_MAXIMA_SACADO", "CONCENTRACAO_MAXIMA_CEDENTE"} {
		if !Conhecido(c) {
			t.Errorf("%s deveria ser conhecido", c)
		}
	}
	if Conhecido("REGRA_X") {
		t.Error("REGRA_X não deveria ser conhecida")
	}
	if !reflect.DeepEqual(len(PorNota), 7) {
		t.Errorf("PorNota tem %d validadores", len(PorNota))
	}
}
