// Compara o motor com o gabarito oficial (data/resultado_esperado_100k.json): o estoque de 100 mil
// linhas avaliado como lote, com as regras de data/validadores_antecipe.json.
package gabarito

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"antecipe/internal/domain/rules"
	"antecipe/internal/repository/filesystem"
	"antecipe/internal/service/engine"
	"antecipe/internal/service/pricing"
)

const dados = "../../../data/"

type esperado struct {
	Linha      int      `json:"linha"`
	ID         string   `json:"id"`
	Status     string   `json:"status"`
	BarradoPor []string `json:"barrado_por"`
}

func semSituacao(cs []string) []string {
	var out []string
	for _, c := range cs {
		if c != "SITUACAO_CADASTRAL_SACADO" {
			out = append(out, c)
		}
	}
	return out
}

func TestGabarito100k(t *testing.T) {
	notas, err := filesystem.LerEstoqueZip(dados + "estoque_antecipe.zip")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(dados + "validadores_antecipe.json")
	reg, err := rules.Ler(b)
	if err != nil {
		t.Fatal(err)
	}
	sit, err := filesystem.LerSituacoes(dados + "cnpj_situacao_mock.csv")
	if err != nil {
		t.Fatal(err)
	}
	var esp []esperado
	eb, _ := os.ReadFile(dados + "resultado_esperado_100k.json")
	if err := json.Unmarshal(eb, &esp); err != nil {
		t.Fatal(err)
	}
	calc, _ := pricing.Nova(pricing.Config{Metodo: pricing.Composto, TaxaMensalPercentual: decimal.NewFromInt(2), BaseDias: 30})
	est := filesystem.ContextoDeEstoque(notas)
	s := engine.Validar(notas, reg, engine.Contexto{
		DataBase: time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC), Situacao: filesystem.SituacaoCom(sit, "ATIVA"),
		Vendidas: est.Vendidas, Estoques: est.Estoques, Itens: est.Itens, Preco: calc,
	})
	if len(s.Resultados) != len(esp) {
		t.Fatalf("linhas = %d, gabarito = %d", len(s.Resultados), len(esp))
	}
	iguais, difSituacao, outras := 0, 0, 0
	for i, e := range esp {
		r := s.Resultados[i]
		if r.Nota.Linha != e.Linha || r.Nota.Numero != e.ID {
			t.Fatalf("linha %d desalinhada: %d %s", e.Linha, r.Nota.Linha, r.Nota.Numero)
		}
		var nossos []string
		for _, x := range r.Recusas {
			nossos = append(nossos, x.Codigo)
		}
		switch {
		case slices.Equal(nossos, e.BarradoPor):
			iguais++
		case slices.Equal(semSituacao(nossos), semSituacao(e.BarradoPor)):
			difSituacao++ // única divergência aceita: o gabarito usou outra fonte de situação cadastral
		default:
			outras++
			if outras <= 5 {
				t.Errorf("linha %d (%s): nosso %v, gabarito %v", e.Linha, e.ID, nossos, e.BarradoPor)
			}
		}
	}
	t.Logf("iguais ao gabarito: %d · só a situação cadastral difere: %d · outras diferenças: %d", iguais, difSituacao, outras)
	if iguais != len(esp) {
		t.Fatalf("esperava 100%% igual ao gabarito: %d de %d (situação %d, outras %d)", iguais, len(esp), difSituacao, outras)
	}
}
