// Testes ponta a ponta: sobem a aplicação inteira (bootstrap + Postgres) e falam com ela por HTTP.
// Precisam de TEST_DATABASE_URL (o schema public é recriado a cada teste).
package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"antecipe/internal/bootstrap"
	"antecipe/internal/platform/observability"
)

const raiz = "../../../"

type app struct {
	srv *httptest.Server
	app *bootstrap.App
	url string
}

func ambiente(t *testing.T) bootstrap.Ambiente {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL não definida")
	}
	return bootstrap.Ambiente{
		DatabaseURL: url, ConfigPath: raiz + "config/antecipe.yaml", DataDir: raiz + "data",
		GestorSenha: "senha-de-teste", TmpDir: t.TempDir(), CORSOrigens: []string{"http://localhost:5173"},
	}
}

func resetar(t *testing.T, url string) {
	t.Helper()
	conn, err := pgx.Connect(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(context.Background())
	if _, err := conn.Exec(context.Background(), "DROP SCHEMA public CASCADE; CREATE SCHEMA public"); err != nil {
		t.Fatal(err)
	}
}

func iniciar(t *testing.T, amb bootstrap.Ambiente, logs io.Writer) *app {
	t.Helper()
	if logs == nil {
		logs = io.Discard
	}
	a, err := bootstrap.Novo(context.Background(), amb, observability.NovoLogger(logs, "info"))
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(a.Handler)
	x := &app{srv: srv, app: a, url: srv.URL}
	t.Cleanup(func() { srv.Close(); a.Fechar(context.Background()) })
	return x
}

func novoApp(t *testing.T) *app {
	amb := ambiente(t)
	resetar(t, amb.DatabaseURL)
	return iniciar(t, amb, nil)
}

func exemplo(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(raiz + "data/exemplo.csv")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func (a *app) enviar(t *testing.T, conteudo []byte, dataBase string) (*http.Response, map[string]any) {
	t.Helper()
	var corpo bytes.Buffer
	mw := multipart.NewWriter(&corpo)
	if dataBase != "" {
		_ = mw.WriteField("data_base", dataBase)
	}
	fw, _ := mw.CreateFormFile("arquivo", "lote.csv")
	_, _ = fw.Write(conteudo)
	_ = mw.Close()
	resp, err := http.Post(a.url+"/api/lotes", mw.FormDataContentType(), &corpo)
	if err != nil {
		t.Fatal(err)
	}
	return resp, lerJSON(t, resp)
}

func (a *app) get(t *testing.T, caminho string) (*http.Response, map[string]any) {
	t.Helper()
	resp, err := http.Get(a.url + caminho)
	if err != nil {
		t.Fatal(err)
	}
	return resp, lerJSON(t, resp)
}

func lerJSON(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	var m map[string]any
	if len(b) > 0 && strings.HasPrefix(resp.Header.Get("Content-Type"), "application/json") {
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatalf("JSON inválido (%d): %s", resp.StatusCode, b)
		}
	}
	return m
}

func (a *app) esperar(t *testing.T, id string) map[string]any {
	t.Helper()
	limite := time.Now().Add(30 * time.Second)
	for time.Now().Before(limite) {
		_, l := a.get(t, "/api/lotes/"+id)
		if s := l["status"]; s == "concluido" || s == "erro" {
			return l
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("lote %s não terminou", id)
	return nil
}

func (a *app) contarLotes(t *testing.T) int {
	t.Helper()
	var n int
	if err := a.app.Repo.Pool.QueryRow(context.Background(), "SELECT count(*) FROM lotes").Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func num(v any) float64 { f, _ := v.(float64); return f }

func TestSaudeEConfig(t *testing.T) {
	a := novoApp(t)
	resp, s := a.get(t, "/api/saude")
	if resp.StatusCode != 200 || s["status"] != "ok" {
		t.Fatalf("saúde = %d %v", resp.StatusCode, s)
	}
	_, c := a.get(t, "/api/config")
	p := c["precificacao"].(map[string]any)
	if p["metodo"] != "DESCONTO_COMPOSTO" || p["descricao"] != "Juros compostos · 2% ao mês" || c["data_base_padrao"] != "2026-10-22" {
		t.Fatalf("config = %v", c)
	}
}

func TestUploadExemploConcluiEMostraProgresso(t *testing.T) {
	a := novoApp(t)
	resp, r := a.enviar(t, exemplo(t), "")
	if resp.StatusCode != http.StatusAccepted || r["lote_id"] == "" || r["replay"] != false {
		t.Fatalf("envio = %d %v", resp.StatusCode, r)
	}
	if !strings.HasPrefix(resp.Header.Get("Location"), "/api/lotes/") {
		t.Fatalf("Location = %q", resp.Header.Get("Location"))
	}
	l := a.esperar(t, r["lote_id"].(string))
	if l["status"] != "concluido" {
		t.Fatalf("lote = %v", l)
	}
	tot := l["totais"].(map[string]any)
	if num(tot["aprovadas"]) < 12 || num(tot["linhas"]) != 23 {
		t.Fatalf("totais = %v", tot)
	}
	etapas := l["etapas"].([]any)
	if len(etapas) != 5 || etapas[4].(map[string]any)["status"] != "concluida" {
		t.Fatalf("etapas = %v", etapas)
	}
	motivos := map[string]bool{}
	for _, m := range l["motivos"].([]any) {
		motivos[m.(map[string]any)["codigo"].(string)] = true
	}
	for _, c := range []string{"DUPLICIDADE", "PRAZO_MAXIMO", "CONCENTRACAO_MAXIMA_SACADO", "FORMATO_INVALIDO"} {
		if !motivos[c] {
			t.Errorf("motivo %s ausente: %v", c, l["motivos"])
		}
	}
	_, n := a.get(t, "/api/lotes/"+r["lote_id"].(string)+"/notas?status=recusada&motivo=PRAZO_MAXIMO")
	itens := n["itens"].([]any)
	if num(n["total"]) != 1 || len(itens) != 1 {
		t.Fatalf("notas = %v", n)
	}
	rec := itens[0].(map[string]any)["recusas"].([]any)[0].(map[string]any)
	if rec["titulo"] != "Prazo acima do limite" || !strings.Contains(rec["mensagem"].(string), "240 dias") {
		t.Fatalf("recusa = %v", rec)
	}
	_, ap := a.get(t, "/api/lotes/"+r["lote_id"].(string)+"/notas?busca=AURORA&status=aprovada")
	item := ap["itens"].([]any)[0].(map[string]any)
	if num(item["liquido_centavos"]) != 980392 || item["calculo"].(map[string]any)["metodo"] != "DESCONTO_COMPOSTO" {
		t.Fatalf("aprovada = %v", item)
	}
	if item["sacado"].(map[string]any)["documento"] != "55.566.677/0001-83" {
		t.Fatalf("documento = %v", item["sacado"])
	}
}

func TestReenvioIdempotente(t *testing.T) {
	a := novoApp(t)
	_, r1 := a.enviar(t, exemplo(t), "")
	a.esperar(t, r1["lote_id"].(string))
	resp, r2 := a.enviar(t, exemplo(t), "")
	if resp.StatusCode != http.StatusOK || r2["replay"] != true || r2["lote_id"] != r1["lote_id"] || resp.Header.Get("Idempotent-Replay") != "true" {
		t.Fatalf("reenvio = %d %v", resp.StatusCode, r2)
	}
	if n := a.contarLotes(t); n != 1 {
		t.Fatalf("lotes = %d", n)
	}
}

func TestEnviosSimultaneos(t *testing.T) {
	a := novoApp(t)
	var wg sync.WaitGroup
	ids := make([]string, 4)
	for i := range ids {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, r := a.enviar(t, exemplo(t), "")
			ids[i], _ = r["lote_id"].(string)
		}(i)
	}
	wg.Wait()
	for _, id := range ids[1:] {
		if id != ids[0] || id == "" {
			t.Fatalf("ids diferentes: %v", ids)
		}
	}
	if n := a.contarLotes(t); n != 1 {
		t.Fatalf("lotes = %d", n)
	}
}

func TestOutraDataBaseGeraOutroLote(t *testing.T) {
	a := novoApp(t)
	_, r1 := a.enviar(t, exemplo(t), "")
	resp, r2 := a.enviar(t, exemplo(t), "2026-10-24")
	if resp.StatusCode != http.StatusAccepted || r2["lote_id"] == r1["lote_id"] {
		t.Fatalf("outra data-base = %d %v", resp.StatusCode, r2)
	}
	_, l := a.get(t, "/api/lotes/"+r2["lote_id"].(string))
	if l["data_base"] != "2026-10-24" {
		t.Fatalf("data_base = %v", l["data_base"])
	}
}

func TestErrosDoArquivo(t *testing.T) {
	a := novoApp(t)
	resp, r := a.enviar(t, []byte("a;b;c\n1;2;3\n"), "")
	if resp.StatusCode != http.StatusUnprocessableEntity || r["erro"].(map[string]any)["codigo"] != "CABECALHO_INVALIDO" {
		t.Fatalf("cabeçalho = %d %v", resp.StatusCode, r)
	}
	resp, r = a.enviar(t, []byte("\x00\x01\x02PNG"), "")
	if resp.StatusCode != http.StatusUnsupportedMediaType || r["erro"].(map[string]any)["codigo"] != "ARQUIVO_NAO_CSV" {
		t.Fatalf("binário = %d %v", resp.StatusCode, r)
	}
	cab, _, _ := bytes.Cut(exemplo(t), []byte("\n"))
	resp, r = a.enviar(t, append(cab, '\n'), "")
	if resp.StatusCode != http.StatusUnprocessableEntity || r["erro"].(map[string]any)["codigo"] != "ARQUIVO_VAZIO" {
		t.Fatalf("só cabeçalho = %d %v", resp.StatusCode, r)
	}
	resp, r = a.enviar(t, exemplo(t), "22/10/2026")
	if resp.StatusCode != http.StatusBadRequest || r["erro"].(map[string]any)["codigo"] != "DATA_BASE_INVALIDA" {
		t.Fatalf("data-base = %d %v", resp.StatusCode, r)
	}
	resp2, err := http.Post(a.url+"/api/lotes", "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	if r := lerJSON(t, resp2); resp2.StatusCode != http.StatusBadRequest || r["erro"].(map[string]any)["codigo"] != "ARQUIVO_AUSENTE" {
		t.Fatalf("sem arquivo = %d %v", resp2.StatusCode, r)
	}
}

func TestLoteInexistente404(t *testing.T) {
	a := novoApp(t)
	for _, c := range []string{"/api/lotes/00000000-0000-0000-0000-000000000000", "/api/lotes/nao-e-uuid"} {
		resp, r := a.get(t, c)
		if resp.StatusCode != http.StatusNotFound || r["erro"].(map[string]any)["codigo"] != "LOTE_NAO_ENCONTRADO" {
			t.Fatalf("%s = %d %v", c, resp.StatusCode, r)
		}
	}
}

func TestRecuperarInterrompidos(t *testing.T) {
	amb := ambiente(t)
	resetar(t, amb.DatabaseURL)
	a := iniciar(t, amb, nil)
	_, r := a.enviar(t, exemplo(t), "")
	id := r["lote_id"].(string)
	a.esperar(t, id)
	if _, err := a.app.Repo.Pool.Exec(context.Background(), "UPDATE lotes SET status = 'processando', etapa = 'validando' WHERE id = $1", id); err != nil {
		t.Fatal(err)
	}
	// a API "reinicia" no meio do lote
	b := iniciar(t, amb, nil)
	_, l := b.get(t, "/api/lotes/"+id)
	if l["status"] != "erro" || l["erro"].(map[string]any)["codigo"] != "PROCESSAMENTO_INTERROMPIDO" {
		t.Fatalf("lote = %v", l)
	}
	resp, r2 := b.enviar(t, exemplo(t), "")
	if resp.StatusCode != http.StatusAccepted || r2["lote_id"] != id {
		t.Fatalf("reenvio = %d %v", resp.StatusCode, r2)
	}
	if l := b.esperar(t, id); l["status"] != "concluido" {
		t.Fatalf("reprocessado = %v", l)
	}
}

func TestValidarJSON(t *testing.T) {
	a := novoApp(t)
	corpo := `{"data_base":"2026-10-22","notas":[{"nome_fundo":"ANTECIPE FIDC","doc_fundo":"92.234.400/0001-80","nome_cedente":"CED","doc_cedente":"10.062.050/0001-57","nome_sacado":"LOJA","doc_sacado":"55.566.677/0001-83","nu_documento":"777001","tipo_recebivel":"Duplicata","valor_nominal":"10000,00","data_vencimento_original":"21/11/2026","data_aquisicao":""}]}`
	resp, err := http.Post(a.url+"/api/validar", "application/json", strings.NewReader(corpo))
	if err != nil {
		t.Fatal(err)
	}
	r := lerJSON(t, resp)
	if resp.StatusCode != 200 {
		t.Fatalf("validar = %d %v", resp.StatusCode, r)
	}
	nota := r["notas"].([]any)[0].(map[string]any)
	if num(nota["liquido_centavos"]) != 980392 || r["precificacao"].(map[string]any)["metodo"] != "DESCONTO_COMPOSTO" {
		t.Fatalf("validar = %v", r)
	}
	// o mesmo endpoint aceita o CSV
	resp, err = http.Post(a.url+"/api/validar", "text/csv", bytes.NewReader(exemplo(t)))
	if err != nil {
		t.Fatal(err)
	}
	r = lerJSON(t, resp)
	if resp.StatusCode != 200 || num(r["totais"].(map[string]any)["linhas"]) != 23 {
		t.Fatalf("validar csv = %d %v", resp.StatusCode, r["totais"])
	}
}

func TestNotasCSV(t *testing.T) {
	a := novoApp(t)
	conteudo := append(exemplo(t), []byte("ANTECIPE FIDC;92.234.400/0001-80;CED;10.062.050/0001-57;=HYPERLINK(\"x\");22.144.961/0001-02;F1;Duplicata;1000,00;21/11/2026;\n")...)
	_, r := a.enviar(t, conteudo, "")
	id := r["lote_id"].(string)
	a.esperar(t, id)
	resp, err := http.Get(a.url + "/api/lotes/" + id + "/notas.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	primeira, _, _ := strings.Cut(string(b), "\n")
	if !strings.HasPrefix(resp.Header.Get("Content-Type"), "text/csv") || !strings.HasSuffix(strings.TrimSpace(primeira), "STATUS;MOTIVO;O_QUE_FAZER;DIAS;DESCONTO;VALOR_LIQUIDO") {
		t.Fatalf("csv = %s / %s", resp.Header.Get("Content-Type"), primeira)
	}
	if strings.Contains(string(b), ";=HYPERLINK") {
		t.Fatal("fórmula não neutralizada no CSV")
	}
	if n := strings.Count(strings.TrimSpace(string(b)), "\n"); n != 24 {
		t.Fatalf("linhas no csv = %d", n)
	}
}

func TestPaginacao(t *testing.T) {
	a := novoApp(t)
	_, r := a.enviar(t, exemplo(t), "")
	id := r["lote_id"].(string)
	a.esperar(t, id)
	_, p := a.get(t, fmt.Sprintf("/api/lotes/%s/notas?por_pagina=5&pagina=2&ordenar=face&direcao=desc", id))
	if num(p["pagina"]) != 2 || num(p["por_pagina"]) != 5 || num(p["total"]) != 23 || num(p["total_paginas"]) != 5 || len(p["itens"].([]any)) != 5 {
		t.Fatalf("paginação = %v", p)
	}
}
