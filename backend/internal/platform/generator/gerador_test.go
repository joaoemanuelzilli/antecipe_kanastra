package generator

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"antecipe/internal/domain"
	"antecipe/internal/domain/rules"
	"antecipe/internal/handler/csvfile"
	"antecipe/internal/repository/filesystem"
	"antecipe/internal/service/engine"
	"antecipe/internal/service/pricing"
)

var dataBase = time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)

func estoque(t *testing.T) []domain.Nota {
	t.Helper()
	n, err := filesystem.LerEstoqueZip("../../../../data/estoque_antecipe.zip")
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func opcoes(linhas int, est []domain.Nota) Opcoes {
	return Opcoes{Linhas: linhas, Semente: 42, DataBase: dataBase, Estoque: est, FundoNome: "ANTECIPE FIDC", FundoCNPJ: "92.234.400/0001-80"}
}

func TestGerarDeterministico(t *testing.T) {
	var a, b bytes.Buffer
	if err := Gerar(&a, opcoes(1000, nil)); err != nil {
		t.Fatal(err)
	}
	if err := Gerar(&b, opcoes(1000, nil)); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Fatal("mesma semente gerou saídas diferentes")
	}
	if n := strings.Count(a.String(), "\n"); n != 1001 {
		t.Fatalf("linhas = %d", n)
	}
}

func TestGerarParseavel(t *testing.T) {
	var buf bytes.Buffer
	if err := Gerar(&buf, opcoes(1000, nil)); err != nil {
		t.Fatal(err)
	}
	notas, err := csvfile.Ler(&buf, nil)
	if err != nil || len(notas) != 1000 {
		t.Fatalf("notas = %d, %v", len(notas), err)
	}
}

func avaliar(t *testing.T, conteudo []byte, est []domain.Nota) engine.Saida {
	t.Helper()
	notas, err := csvfile.Ler(bytes.NewReader(conteudo), nil)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile("../../../../data/validadores_antecipe.json")
	r, _ := rules.Ler(b)
	sit, err := filesystem.LerSituacoes("../../../../data/cnpj_situacao_mock.csv")
	if err != nil {
		t.Fatal(err)
	}
	calc, _ := pricing.Nova(pricing.Config{Metodo: pricing.Composto, TaxaMensalPercentual: decimal.NewFromInt(2), BaseDias: 30})
	c := filesystem.ContextoDeEstoque(est)
	return engine.Validar(notas, r, engine.Contexto{
		DataBase: dataBase, Situacao: filesystem.SituacaoCom(sit, "ATIVA"), Vendidas: c.Vendidas, Estoques: c.Estoques, Itens: c.Itens, Preco: calc,
	})
}

func TestGerarMistura(t *testing.T) {
	est := estoque(t)
	var buf bytes.Buffer
	if err := Gerar(&buf, opcoes(20000, est)); err != nil {
		t.Fatal(err)
	}
	s := avaliar(t, buf.Bytes(), est)
	for _, c := range []string{"DUPLICADA_NO_ARQUIVO", "FORMATO_INVALIDO", "PRAZO_MAXIMO", "NAO_VENCIDA", "VALOR_MINIMO", "TIPO_RECEBIVEL", "CNPJ_SACADO_VALIDO", "SITUACAO_CADASTRAL_SACADO"} {
		if s.Motivos[c] == 0 {
			t.Errorf("nenhuma recusa %s (motivos %v)", c, s.Motivos)
		}
	}
	pct := float64(s.Totais.Aprovadas) * 100 / float64(s.Totais.Linhas)
	if pct < 70 || pct > 90 {
		t.Errorf("aprovadas = %.1f%%", pct)
	}
}

func TestExemploCobreTodosOsCodigos(t *testing.T) {
	est := estoque(t)
	var buf bytes.Buffer
	if err := Exemplo(&buf, dataBase, est); err != nil {
		t.Fatal(err)
	}
	s := avaliar(t, buf.Bytes(), est)
	for _, c := range []string{"CNPJ_SACADO_VALIDO", "SITUACAO_CADASTRAL_SACADO", "TIPO_RECEBIVEL", "VALOR_MINIMO", "NAO_VENCIDA", "PRAZO_MAXIMO", "CONCENTRACAO_MAXIMA_SACADO", "DUPLICADA_NO_ARQUIVO", "FORMATO_INVALIDO"} {
		if s.Motivos[c] == 0 {
			t.Errorf("exemplo sem recusa %s (motivos %v)", c, s.Motivos)
		}
	}
	if s.Totais.Aprovadas < 10 {
		t.Errorf("aprovadas = %d", s.Totais.Aprovadas)
	}
	achou := false
	for _, r := range s.Resultados {
		if r.Nota.NomeSacado == "LOJA AURORA LTDA" && r.Status == domain.Aprovada && r.LiquidoCentavos == 980392 {
			achou = true
		}
	}
	if !achou {
		t.Error("exemplo sem a nota de R$ 10.000,00 a 30 dias aprovada por R$ 9.803,92")
	}
}
