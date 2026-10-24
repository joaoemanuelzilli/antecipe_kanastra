package filesystem

import (
	"testing"

	"antecipe/internal/platform/document"
)

const arquivoZip = "../../../../data/estoque_antecipe.zip"

func TestLerEstoqueZip(t *testing.T) {
	notas, err := LerEstoqueZip(arquivoZip)
	if err != nil {
		t.Fatal(err)
	}
	if len(notas) != 100554 {
		t.Fatalf("notas = %d", len(notas))
	}
	antecipe := 0
	for i := range notas {
		if document.Normalizar(notas[i].DocFundo) == "92234400000180" {
			antecipe++
		}
		if notas[i].ErroFormato != nil {
			t.Fatalf("linha %d com erro: %+v", notas[i].Linha, notas[i].ErroFormato)
		}
	}
	if antecipe != 10621 {
		t.Fatalf("ANTECIPE = %d", antecipe)
	}
}

func TestContextoDeEstoque(t *testing.T) {
	notas, err := LerEstoqueZip(arquivoZip)
	if err != nil {
		t.Fatal(err)
	}
	c := ContextoDeEstoque(notas)
	if len(c.Itens) != 100554 || len(c.Vendidas) != 0 || len(c.Estoques) != 18 {
		t.Fatalf("itens = %d, vendidas = %d, fundos = %d", len(c.Itens), len(c.Vendidas), len(c.Estoques))
	}
	est := c.Estoques["92234400000180"]
	if est.TotalCentavos != 5307402800 {
		t.Fatalf("total = %d", est.TotalCentavos)
	}
	var raiz int64
	for doc, v := range est.PorSacado {
		if document.Raiz(doc) == "58206002" {
			raiz += v
		}
	}
	pct := float64(raiz) * 100 / float64(est.TotalCentavos)
	if pct < 12.49 || pct > 12.51 {
		t.Fatalf("concentração 58206002 = %.3f%%", pct)
	}
}

func TestLerSituacoes(t *testing.T) {
	m, err := LerSituacoes("../../../../data/cnpj_situacao_mock.csv")
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 4360 || m["08356446000165"] != "INAPTA" || m["09362675000236"] != "BAIXADA" || m["03601302000103"] != "SUSPENSA" {
		t.Fatalf("situações = %d", len(m))
	}
	f := SituacaoCom(m, "ATIVA")
	if f("08356446000165") != "INAPTA" || f("99999999000199") != "ATIVA" {
		t.Fatal("SituacaoCom não aplica o padrão")
	}
}
