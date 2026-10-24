package e2e

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"antecipe/internal/domain"
	"antecipe/internal/platform/generator"
)

func (a *app) ceder(t *testing.T, id, chave string) (*http.Response, map[string]any) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, a.url+"/api/lotes/"+id+"/cessao", nil)
	if chave != "" {
		req.Header.Set("Idempotency-Key", chave)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp, lerJSON(t, resp)
}

func (a *app) contarCedidas(t *testing.T) int {
	t.Helper()
	var n int
	if err := a.app.Repo.Pool.QueryRow(context.Background(), "SELECT count(*) FROM estoque WHERE origem = 'cessao'").Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

// chavesAprovadas lista as chaves das notas aprovadas de um lote, pela API.
func (a *app) chavesAprovadas(t *testing.T, id string) map[string]int {
	t.Helper()
	out := map[string]int{}
	for pagina := 1; ; pagina++ {
		_, p := a.get(t, fmt.Sprintf("/api/lotes/%s/notas?status=aprovada&por_pagina=200&pagina=%d", id, pagina))
		itens := p["itens"].([]any)
		for _, x := range itens {
			m := x.(map[string]any)
			venc, _ := time.Parse("2006-01-02", m["vencimento"].(string))
			n := domain.Nota{DocCedente: m["cedente"].(map[string]any)["documento"].(string), Numero: m["numero"].(string), Vencimento: venc}
			out[domain.Chave(&n)] = int(num(m["linha"]))
		}
		if pagina >= int(num(p["total_paginas"])) {
			return out
		}
	}
}

func gerar(t *testing.T, linhas int, semente int64) []byte {
	t.Helper()
	var b bytes.Buffer
	err := generator.Gerar(&b, generator.Opcoes{Linhas: linhas, Semente: semente, DataBase: time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC), FundoNome: "ANTECIPE FIDC", FundoCNPJ: "92.234.400/0001-80"})
	if err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func TestCessaoReplay(t *testing.T) {
	a := novoApp(t)
	_, r := a.enviar(t, exemplo(t), "")
	id := r["lote_id"].(string)
	l := a.esperar(t, id)
	aprovadas := int(num(l["totais"].(map[string]any)["aprovadas"]))

	resp, c := a.ceder(t, id, "chave-1")
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("cessão = %d %v", resp.StatusCode, c)
	}
	ces := c["cessao"].(map[string]any)
	// a última nota do exemplo já está no estoque: aprovada na simulação, fica fora da venda
	exc := c["notas_excluidas"].([]any)
	if int(num(ces["notas_cedidas"])) != aprovadas-1 || len(exc) != 1 || exc[0].(map[string]any)["codigo"] != "DUPLICIDADE" || num(exc[0].(map[string]any)["linha"]) != 24 {
		t.Fatalf("cessão = %v, excluídas = %v (aprovadas %d)", ces, exc, aprovadas)
	}
	resp, c2 := a.ceder(t, id, "chave-1")
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Idempotent-Replay") != "true" || c2["cessao"].(map[string]any)["id"] != ces["id"] {
		t.Fatalf("replay = %d %v", resp.StatusCode, c2)
	}
	if n := a.contarCedidas(t); n != aprovadas-1 {
		t.Fatalf("cedidas = %d, esperado %d", n, aprovadas-1)
	}
	_, l2 := a.get(t, "/api/lotes/"+id)
	if l2["cessao"] == nil {
		t.Fatal("lote não mostra a cessão")
	}
	// a mesma Idempotency-Key em outro lote é recusada
	_, r2 := a.enviar(t, gerar(t, 50, 9), "")
	a.esperar(t, r2["lote_id"].(string))
	resp, c3 := a.ceder(t, r2["lote_id"].(string), "chave-1")
	if resp.StatusCode != http.StatusConflict || c3["erro"].(map[string]any)["codigo"] != "IDEMPOTENCY_KEY_REUTILIZADA" {
		t.Fatalf("chave reutilizada = %d %v", resp.StatusCode, c3)
	}
}

func TestCessoesSimultaneasNaoVendemDuasVezes(t *testing.T) {
	a := novoApp(t)
	loteA := gerar(t, 200, 1)
	linhasA := strings.SplitAfter(string(loteA), "\n")
	loteB := []byte(strings.Join(linhasA[:21], "") + strings.Join(strings.SplitAfter(string(gerar(t, 200, 2)), "\n")[1:], ""))

	_, ra := a.enviar(t, loteA, "")
	_, rb := a.enviar(t, loteB, "")
	idA, idB := ra["lote_id"].(string), rb["lote_id"].(string)
	a.esperar(t, idA)
	a.esperar(t, idB)
	aprovA, aprovB := a.chavesAprovadas(t, idA), a.chavesAprovadas(t, idB)
	uniao := map[string]bool{}
	comuns := 0
	for k := range aprovA {
		uniao[k] = true
	}
	for k := range aprovB {
		if uniao[k] {
			comuns++
		}
		uniao[k] = true
	}
	if comuns == 0 {
		t.Fatal("cenário sem notas em comum")
	}

	var wg sync.WaitGroup
	res := make([]map[string]any, 2)
	codigos := make([]int, 2)
	for i, id := range []string{idA, idB} {
		wg.Add(1)
		go func(i int, id string) {
			defer wg.Done()
			resp, c := a.ceder(t, id, "")
			codigos[i], res[i] = resp.StatusCode, c
		}(i, id)
	}
	wg.Wait()
	if codigos[0] != http.StatusCreated || codigos[1] != http.StatusCreated {
		t.Fatalf("códigos = %v %v", codigos, res)
	}
	cedidas := int(num(res[0]["cessao"].(map[string]any)["notas_cedidas"])) + int(num(res[1]["cessao"].(map[string]any)["notas_cedidas"]))
	excluidasDup := 0
	for _, r := range res {
		for _, e := range r["notas_excluidas"].([]any) {
			if e.(map[string]any)["codigo"] == "DUPLICIDADE" {
				excluidasDup++
			}
		}
	}
	if excluidasDup != comuns {
		t.Fatalf("excluídas por duplicidade = %d, notas em comum = %d", excluidasDup, comuns)
	}
	if n := a.contarCedidas(t); n != len(uniao) || cedidas != len(uniao) {
		t.Fatalf("cedidas no estoque = %d, na resposta = %d, união = %d", n, cedidas, len(uniao))
	}
}

func TestCessaoErros(t *testing.T) {
	a := novoApp(t)
	resp, c := a.ceder(t, "00000000-0000-0000-0000-000000000000", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("inexistente = %d %v", resp.StatusCode, c)
	}
	_, r := a.enviar(t, exemplo(t), "")
	id := r["lote_id"].(string)
	a.esperar(t, id)
	if _, err := a.app.Repo.Pool.Exec(context.Background(), "UPDATE lotes SET status = 'processando' WHERE id = $1", id); err != nil {
		t.Fatal(err)
	}
	resp, c = a.ceder(t, id, "")
	if resp.StatusCode != http.StatusConflict || c["erro"].(map[string]any)["codigo"] != "LOTE_NAO_CONCLUIDO" {
		t.Fatalf("processando = %d %v", resp.StatusCode, c)
	}
	// lote só com recusas
	linhas := strings.Split(strings.TrimSpace(string(exemplo(t))), "\n")
	soRecusas := linhas[0] + "\n" + strings.Join(linhas[13:20], "\n") + "\n"
	_, r2 := a.enviar(t, []byte(soRecusas), "")
	id2 := r2["lote_id"].(string)
	if l := a.esperar(t, id2); num(l["totais"].(map[string]any)["aprovadas"]) != 0 {
		t.Fatalf("esperava zero aprovadas: %v", l["totais"])
	}
	resp, c = a.ceder(t, id2, "")
	if resp.StatusCode != http.StatusConflict || c["erro"].(map[string]any)["codigo"] != "SEM_APROVADAS" {
		t.Fatalf("sem aprovadas = %d %v", resp.StatusCode, c)
	}
}

func TestCessaoChaveDeOutroLoteNaoBloqueia(t *testing.T) {
	a := novoApp(t)
	_, ra := a.enviar(t, gerar(t, 60, 31), "")
	_, rb := a.enviar(t, gerar(t, 60, 32), "")
	idA, idB := ra["lote_id"].(string), rb["lote_id"].(string)
	a.esperar(t, idA)
	a.esperar(t, idB)
	if resp, c := a.ceder(t, idA, "lote:"+idB); resp.StatusCode != http.StatusCreated {
		t.Fatalf("cessão A = %d %v", resp.StatusCode, c)
	}
	if resp, c := a.ceder(t, idB, ""); resp.StatusCode != http.StatusCreated {
		t.Fatalf("cessão B bloqueada pela chave escolhida em A = %d %v", resp.StatusCode, c)
	}
}
