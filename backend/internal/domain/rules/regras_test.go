package rules

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/shopspring/decimal"
)

func original(t *testing.T) Regras {
	t.Helper()
	b, err := os.ReadFile("../../../../data/validadores_antecipe.json")
	if err != nil {
		t.Fatal(err)
	}
	r, err := Ler(b)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func achar(r Regras, codigo string) *Validador {
	for i := range r.Validadores {
		if r.Validadores[i].Codigo == codigo {
			return &r.Validadores[i]
		}
	}
	return nil
}

func TestLerJSONOriginal(t *testing.T) {
	r := original(t)
	if len(r.Validadores) != 9 {
		t.Fatalf("validadores = %d", len(r.Validadores))
	}
	ativos := 0
	for _, v := range r.Validadores {
		if v.Ativo {
			ativos++
		}
	}
	if ativos != 7 {
		t.Fatalf("ativos = %d", ativos)
	}
	if r.Fundo.Nome != "ANTECIPE FIDC" || r.Fundo.CNPJ != "92.234.400/0001-80" {
		t.Fatalf("fundo = %+v", r.Fundo)
	}
	p := r.Precificacao
	if p.Metodo != "DESCONTO_SIMPLES" || !p.TaxaMensalPercentual.Equal(decimal.NewFromInt(2)) || p.BaseDias != 30 {
		t.Fatalf("precificacao = %+v", p)
	}
}

func TestParametros(t *testing.T) {
	r := original(t)
	if n, err := achar(r, "PRAZO_MAXIMO").Parametros.Inteiro("prazo_maximo_dias"); err != nil || n != 180 {
		t.Fatalf("prazo = %d, %v", n, err)
	}
	if d, err := achar(r, "VALOR_MINIMO").Parametros.Decimal("valor_minimo"); err != nil || !d.Equal(decimal.NewFromInt(100)) {
		t.Fatalf("valor_minimo = %s, %v", d, err)
	}
	if l, err := achar(r, "TIPO_RECEBIVEL").Parametros.ListaTexto("tipos_aceitos"); err != nil || len(l) != 2 || l[1] != "Duplicata de Serviço" {
		t.Fatalf("tipos = %v, %v", l, err)
	}
	if s, err := achar(r, "CONCENTRACAO_MAXIMA_SACADO").Parametros.Texto("agrupamento"); err != nil || s != "RAIZ_CNPJ" {
		t.Fatalf("agrupamento = %q, %v", s, err)
	}
	if _, err := achar(r, "TIPO_RECEBIVEL").Parametros.Inteiro("tipos_aceitos"); err == nil {
		t.Fatal("Inteiro numa lista deveria falhar")
	}
	if _, err := achar(r, "PRAZO_MAXIMO").Parametros.Inteiro("nao_existe"); err == nil {
		t.Fatal("parâmetro ausente deveria falhar")
	}
	p := Parametros{"x": json.RawMessage(`180.5`)}
	if _, err := p.Inteiro("x"); err == nil {
		t.Fatal("180.5 não é inteiro")
	}
}

func TestValidarSchemaOriginalSemErros(t *testing.T) {
	if errs := ValidarSchema(original(t), []string{"DESCONTO_SIMPLES", "DESCONTO_COMPOSTO"}); len(errs) != 0 {
		t.Fatalf("erros = %+v", errs)
	}
}

func TestValidarSchemaRecusa(t *testing.T) {
	r := original(t)
	achar(r, "PRAZO_MAXIMO").Parametros["prazo_maximo_dias"] = json.RawMessage(`-1`)
	errs := ValidarSchema(r, []string{"DESCONTO_SIMPLES"})
	if len(errs) != 1 || errs[0].Campo != "validadores[5].parametros.prazo_maximo_dias" {
		t.Fatalf("erros = %+v", errs)
	}
}

func TestValidarSchemaVarios(t *testing.T) {
	r := original(t)
	r.Validadores = append(r.Validadores, Validador{Codigo: "REGRA_X", Ativo: true, Parametros: Parametros{}})
	achar(r, "CONCENTRACAO_MAXIMA_SACADO").Parametros["percentual_maximo"] = json.RawMessage(`150`)
	achar(r, "CONCENTRACAO_MAXIMA_SACADO").Parametros["agrupamento"] = json.RawMessage(`"FILIAL"`)
	achar(r, "TIPO_RECEBIVEL").Parametros["tipos_aceitos"] = json.RawMessage(`[]`)
	r.Precificacao.Metodo = "MAGICO"
	r.Precificacao.BaseDias = 0
	r.Fundo.CNPJ = "123"
	campos := map[string]bool{}
	for _, e := range ValidarSchema(r, []string{"DESCONTO_SIMPLES"}) {
		campos[e.Campo] = true
	}
	for _, c := range []string{
		"validadores[9].codigo",
		"validadores[6].parametros.percentual_maximo",
		"validadores[6].parametros.agrupamento",
		"validadores[2].parametros.tipos_aceitos",
		"precificacao.metodo",
		"precificacao.base_dias",
		"fundo.cnpj",
	} {
		if !campos[c] {
			t.Errorf("faltou erro em %s (achados: %v)", c, campos)
		}
	}
}

func TestResumo(t *testing.T) {
	antes := original(t)
	depois := original(t)
	achar(depois, "PRAZO_MAXIMO").Parametros["prazo_maximo_dias"] = json.RawMessage(`120`)
	got := Resumo(antes, depois)
	if len(got) != 1 || got[0] != "PRAZO_MAXIMO: 180 → 120 dias" {
		t.Fatalf("resumo = %v", got)
	}

	depois2 := original(t)
	achar(depois2, "PRAZO_MINIMO").Ativo = true
	achar(depois2, "TIPO_RECEBIVEL").Parametros["tipos_aceitos"] = json.RawMessage(`["Duplicata","Duplicata de Serviço","Cheque"]`)
	depois2.Precificacao.TaxaMensalPercentual = decimal.RequireFromString("2.5")
	got = Resumo(antes, depois2)
	esperado := []string{
		"TIPO_RECEBIVEL: Duplicata e Duplicata de Serviço → Duplicata, Duplicata de Serviço e Cheque",
		"PRAZO_MINIMO: ligada",
		"Taxa: 2% → 2,5% ao mês",
	}
	if strings.Join(got, "|") != strings.Join(esperado, "|") {
		t.Fatalf("resumo = %v", got)
	}
}

func TestJSONTaxaComoNumero(t *testing.T) {
	b, err := json.Marshal(original(t).Precificacao)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"taxa_mensal_percentual":2`) {
		t.Fatalf("json = %s", b)
	}
}

func TestCatalogo(t *testing.T) {
	c := Catalogo()
	if len(c) != 9 {
		t.Fatalf("catálogo = %d", len(c))
	}
	for _, item := range c {
		if item.Titulo == "" || item.Descricao == "" {
			t.Errorf("item sem título/descrição: %+v", item)
		}
	}
	if len(CatalogoPrecificacao()) != 2 {
		t.Fatal("catálogo de precificação deve ter taxa e base")
	}
}

func TestParametroNumeroExtremo(t *testing.T) {
	for _, v := range []string{`1e300`, `-1e300`, `1e-300`, `12345678901234567890123456789012345`, `9223372036854775808`} {
		p := Parametros{"x": json.RawMessage(v)}
		if _, err := p.Inteiro("x"); err == nil {
			t.Errorf("Inteiro(%s) deveria falhar", v)
		}
		if _, err := p.Decimal("x"); err == nil {
			t.Errorf("Decimal(%s) deveria falhar", v)
		}
	}
	p := Parametros{"x": json.RawMessage(`3650`)}
	if n, err := p.Inteiro("x"); err != nil || n != 3650 {
		t.Fatalf("3650 = %d, %v", n, err)
	}
}
