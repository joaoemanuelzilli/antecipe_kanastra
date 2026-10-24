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
	vendidas, est := ContextoDeEstoque(notas, "92.234.400/0001-80")
	if len(vendidas) != 100554 {
		t.Fatalf("vendidas = %d", len(vendidas))
	}
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
	m, err := LerSituacoes("../../../../data/situacao_cadastral.csv")
	if err != nil {
		t.Fatal(err)
	}
	if len(m) != 6 || m["11122233000183"] != "INAPTA" || m["22233344000183"] != "BAIXADA" {
		t.Fatalf("situações = %v", m)
	}
	f := SituacaoCom(m, "ATIVA")
	if f("11122233000183") != "INAPTA" || f("99999999000199") != "ATIVA" {
		t.Fatal("SituacaoCom não aplica o padrão")
	}
}
