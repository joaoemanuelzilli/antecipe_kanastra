package e2e

import (
	"archive/zip"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"slices"
	"testing"
)

type itemGabarito struct {
	Linha      int      `json:"linha"`
	ID         string   `json:"id"`
	Status     string   `json:"status"`
	BarradoPor []string `json:"barrado_por"`
}

func (a *app) resultado(t *testing.T, id string) []itemGabarito {
	t.Helper()
	resp, err := http.Get(a.url + "/api/lotes/" + id + "/resultado.json")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("resultado.json = %d", resp.StatusCode)
	}
	var out []itemGabarito
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestResultadoNoFormatoDoGabarito(t *testing.T) {
	a := novoApp(t)
	_, r := a.enviar(t, exemplo(t), "")
	id := r["lote_id"].(string)
	a.esperar(t, id)
	res := a.resultado(t, id)
	if len(res) != 23 || res[0].Linha != 2 || res[0].Status != "APROVADO" || len(res[0].BarradoPor) != 0 {
		t.Fatalf("resultado = %v", res[:2])
	}
	if res[12].Status != "BARRADO" || res[12].BarradoPor[0] != "CNPJ_SACADO_VALIDO" {
		t.Fatalf("linha 14 = %+v", res[12])
	}
}

// TestGabaritoPelaAPI envia o estoque de 100 mil linhas (o arquivo dos jurados) e compara com o gabarito.
func TestGabaritoPelaAPI(t *testing.T) {
	a := novoApp(t)
	zr, err := zip.OpenReader(raiz + "data/estoque_antecipe.zip")
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	f, _ := zr.File[0].Open()
	conteudo, _ := io.ReadAll(f)
	f.Close()

	_, r := a.enviar(t, conteudo, "")
	id := r["lote_id"].(string)
	if l := a.esperar(t, id); l["status"] != "concluido" {
		t.Fatalf("lote = %v", l)
	}
	nossos := a.resultado(t, id)
	var esp []itemGabarito
	b, _ := os.ReadFile(raiz + "data/resultado_esperado_100k.json")
	if err := json.Unmarshal(b, &esp); err != nil {
		t.Fatal(err)
	}
	if len(nossos) != len(esp) {
		t.Fatalf("linhas = %d, gabarito = %d", len(nossos), len(esp))
	}
	sem := func(cs []string) []string {
		var out []string
		for _, c := range cs {
			if c != "SITUACAO_CADASTRAL_SACADO" {
				out = append(out, c)
			}
		}
		return out
	}
	iguais, situacao, outras := 0, 0, 0
	for i := range esp {
		n, e := nossos[i], esp[i]
		switch {
		case n.Linha != e.Linha || n.ID != e.ID:
			t.Fatalf("linha %d desalinhada: %+v", e.Linha, n)
		case n.Status == e.Status && slices.Equal(n.BarradoPor, e.BarradoPor):
			iguais++
		case slices.Equal(sem(n.BarradoPor), sem(e.BarradoPor)):
			situacao++
		default:
			outras++
			if outras <= 3 {
				t.Errorf("linha %d: nosso %+v, gabarito %+v", e.Linha, n, e)
			}
		}
	}
	t.Logf("pela API: %d iguais ao gabarito, %d diferem só na situação cadastral, %d outras", iguais, situacao, outras)
	if outras > 0 || iguais < 97000 {
		t.Fatalf("divergências fora da situação cadastral: %d (iguais %d)", outras, iguais)
	}
}
