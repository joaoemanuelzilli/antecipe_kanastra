package e2e

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"testing"
	"time"
)

// cliente com cookies próprios (cada um é um navegador diferente)
func navegador(t *testing.T) *http.Client {
	t.Helper()
	jar, _ := cookiejar.New(nil)
	return &http.Client{Jar: jar}
}

func (a *app) postJSON(t *testing.T, c *http.Client, caminho string, corpo any) (*http.Response, map[string]any) {
	t.Helper()
	b, _ := json.Marshal(corpo)
	resp, err := c.Post(a.url+caminho, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	return resp, lerJSON(t, resp)
}

func (a *app) getCom(t *testing.T, c *http.Client, caminho string) (*http.Response, map[string]any) {
	t.Helper()
	resp, err := c.Get(a.url + caminho)
	if err != nil {
		t.Fatal(err)
	}
	return resp, lerJSON(t, resp)
}

func (a *app) enviarCom(t *testing.T, c *http.Client, conteudo []byte) map[string]any {
	t.Helper()
	var corpo bytes.Buffer
	mw := multipart.NewWriter(&corpo)
	fw, _ := mw.CreateFormFile("arquivo", "lote.csv")
	_, _ = fw.Write(conteudo)
	_ = mw.Close()
	resp, err := c.Post(a.url+"/api/lotes", mw.FormDataContentType(), &corpo)
	if err != nil {
		t.Fatal(err)
	}
	return lerJSON(t, resp)
}

func cadastro(email string) map[string]any {
	return map[string]any{"nome": "Ana Souza", "email": email, "senha": "senha-forte-123", "empresa_nome": "Ana Estofados LTDA", "empresa_cnpj": "10.062.050/0001-57"}
}

func TestCadastroLoginELogout(t *testing.T) {
	a := novoApp(t)
	c := navegador(t)
	resp, r := a.postJSON(t, c, "/api/auth/cadastro", cadastro("Ana@Exemplo.com"))
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("cadastro = %d %v", resp.StatusCode, r)
	}
	u := r["usuario"].(map[string]any)
	if u["perfil"] != "empresa" || u["email"] != "ana@exemplo.com" || u["senha"] != nil || u["senha_hash"] != nil {
		t.Fatalf("usuário = %v", u)
	}
	ck := resp.Header.Get("Set-Cookie")
	if !strings.Contains(ck, "antecipe_sessao=") || !strings.Contains(ck, "HttpOnly") || !strings.Contains(ck, "SameSite=Lax") {
		t.Fatalf("cookie = %q", ck)
	}
	if resp, eu := a.getCom(t, c, "/api/auth/eu"); resp.StatusCode != 200 || eu["usuario"].(map[string]any)["nome"] != "Ana Souza" {
		t.Fatalf("eu = %d %v", resp.StatusCode, eu)
	}
	if resp, _ := a.postJSON(t, c, "/api/auth/logout", nil); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("logout = %d", resp.StatusCode)
	}
	if resp, _ := a.getCom(t, c, "/api/auth/eu"); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("eu depois do logout = %d", resp.StatusCode)
	}
	// login de novo (e-mail sem diferenciar maiúsculas); o token também serve como Bearer
	resp, l := a.postJSON(t, navegador(t), "/api/auth/login", map[string]any{"email": "ANA@exemplo.com", "senha": "senha-forte-123"})
	if resp.StatusCode != 200 || l["token"] == "" {
		t.Fatalf("login = %d %v", resp.StatusCode, l)
	}
	req, _ := http.NewRequest(http.MethodGet, a.url+"/api/auth/eu", nil)
	req.Header.Set("Authorization", "Bearer "+l["token"].(string))
	if resp, err := http.DefaultClient.Do(req); err != nil || resp.StatusCode != 200 {
		t.Fatalf("bearer = %v %v", resp, err)
	}
}

func TestCadastroValida(t *testing.T) {
	a := novoApp(t)
	casos := []struct {
		corpo  map[string]any
		status int
		codigo string
	}{
		{func() map[string]any { m := cadastro("x@y.com"); m["senha"] = "curta"; return m }(), 422, "CADASTRO_INVALIDO"},
		{func() map[string]any { m := cadastro("x@y.com"); m["empresa_cnpj"] = "77.788.899/0001-80"; return m }(), 422, "CADASTRO_INVALIDO"},
		{func() map[string]any { m := cadastro("nao-e-email"); return m }(), 422, "CADASTRO_INVALIDO"},
		{func() map[string]any { m := cadastro("x@y.com"); m["perfil"] = "gestor"; return m }(), 201, ""}, // perfil é ignorado: sempre empresa
		{cadastro("x@y.com"), 409, "EMAIL_JA_CADASTRADO"},
	}
	for i, k := range casos {
		resp, r := a.postJSON(t, navegador(t), "/api/auth/cadastro", k.corpo)
		if resp.StatusCode != k.status {
			t.Fatalf("caso %d: %d %v", i, resp.StatusCode, r)
		}
		if k.codigo != "" && r["erro"].(map[string]any)["codigo"] != k.codigo {
			t.Fatalf("caso %d: %v", i, r)
		}
		if k.status == 201 && r["usuario"].(map[string]any)["perfil"] != "empresa" {
			t.Fatalf("cadastro virou gestor: %v", r)
		}
	}
	if resp, r := a.postJSON(t, navegador(t), "/api/auth/login", map[string]any{"email": "x@y.com", "senha": "errada-123456"}); resp.StatusCode != 401 || r["erro"].(map[string]any)["codigo"] != "CREDENCIAIS_INVALIDAS" {
		t.Fatalf("senha errada = %d %v", resp.StatusCode, r)
	}
	if resp, _ := a.postJSON(t, navegador(t), "/api/auth/login", map[string]any{"email": "ninguem@y.com", "senha": "qualquer-123"}); resp.StatusCode != 401 {
		t.Fatalf("e-mail inexistente = %d", resp.StatusCode)
	}
}

func TestGestorDoEnvECriacaoDeGestor(t *testing.T) {
	a := novoApp(t)
	g := navegador(t)
	resp, r := a.postJSON(t, g, "/api/auth/login", map[string]any{"email": "gestor@antecipe.local", "senha": "senha-de-teste"})
	if resp.StatusCode != 200 || r["usuario"].(map[string]any)["perfil"] != "gestor" {
		t.Fatalf("login do gestor = %d %v", resp.StatusCode, r)
	}
	// a sessão de gestor libera as rotas do gestor sem o header antigo
	req, _ := http.NewRequest(http.MethodPut, a.url+"/api/regras", bytes.NewReader(mustJSON(map[string]any{"regras": a.regrasCom(t, 120), "autor": "Rafael"})))
	req.Header.Set("Content-Type", "application/json")
	if resp, err := g.Do(req); err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("PUT com sessão de gestor = %v %v", resp.StatusCode, err)
	}
	if resp, r := a.postJSON(t, g, "/api/gestor/usuarios", map[string]any{"nome": "Rafael", "email": "rafael@fundo.com", "senha": "outra-senha-123"}); resp.StatusCode != http.StatusCreated || r["usuario"].(map[string]any)["perfil"] != "gestor" {
		t.Fatalf("criar gestor = %d %v", resp.StatusCode, r)
	}
	if resp, _ := a.postJSON(t, navegador(t), "/api/auth/login", map[string]any{"email": "rafael@fundo.com", "senha": "outra-senha-123"}); resp.StatusCode != 200 {
		t.Fatalf("login do gestor novo = %d", resp.StatusCode)
	}
	// empresa não vira gestor nem acessa rotas do gestor
	e := navegador(t)
	a.postJSON(t, e, "/api/auth/cadastro", cadastro("emp@x.com"))
	if resp, _ := a.postJSON(t, e, "/api/gestor/usuarios", map[string]any{"nome": "X", "email": "x2@x.com", "senha": "senha-123456"}); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("empresa criando gestor = %d", resp.StatusCode)
	}
	if resp, _ := a.getCom(t, e, "/api/gestor/verificar"); resp.StatusCode != http.StatusForbidden {
		t.Fatalf("empresa no verificar = %d", resp.StatusCode)
	}
}

func (a *app) esperarCom(t *testing.T, c *http.Client, id string) map[string]any {
	t.Helper()
	for i := 0; i < 600; i++ {
		_, l := a.getCom(t, c, "/api/lotes/"+id)
		if s := l["status"]; s == "concluido" || s == "erro" {
			return l
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("lote %s não terminou", id)
	return nil
}

func TestLotesPorUsuario(t *testing.T) {
	a := novoApp(t)
	ana, bia, gestor := navegador(t), navegador(t), navegador(t)
	a.postJSON(t, ana, "/api/auth/cadastro", cadastro("ana@x.com"))
	a.postJSON(t, bia, "/api/auth/cadastro", cadastro("bia@x.com"))
	a.postJSON(t, gestor, "/api/auth/login", map[string]any{"email": "gestor@antecipe.local", "senha": "senha-de-teste"})

	id := a.enviarCom(t, ana, exemplo(t))["lote_id"].(string)
	if l := a.esperarCom(t, ana, id); l["status"] != "concluido" {
		t.Fatalf("lote da Ana = %v", l)
	}
	// o mesmo arquivo enviado pela Bia vira outro lote, dela
	idBia := a.enviarCom(t, bia, exemplo(t))["lote_id"].(string)
	if idBia == id {
		t.Fatal("lote da Bia reaproveitou o lote da Ana")
	}
	a.esperarCom(t, bia, idBia)

	_, lista := a.getCom(t, ana, "/api/lotes")
	itens := lista["lotes"].([]any)
	if len(itens) != 1 || itens[0].(map[string]any)["id"] != id {
		t.Fatalf("lotes da Ana = %v", lista)
	}
	_, todos := a.getCom(t, gestor, "/api/lotes")
	if len(todos["lotes"].([]any)) != 2 {
		t.Fatalf("lotes vistos pelo gestor = %v", todos)
	}
	if resp, _ := a.get(t, "/api/lotes"); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("lista sem login = %d", resp.StatusCode)
	}
	// ninguém além da dona (e do gestor) vê o lote, nem as notas, nem baixa nada
	for _, rota := range []string{"/api/lotes/" + id, "/api/lotes/" + id + "/notas", "/api/lotes/" + id + "/notas.csv", "/api/lotes/" + id + "/resultado.json"} {
		if resp, _ := a.getCom(t, bia, rota); resp.StatusCode != http.StatusNotFound {
			t.Fatalf("Bia em %s = %d", rota, resp.StatusCode)
		}
		if resp, _ := a.get(t, rota); resp.StatusCode != http.StatusNotFound {
			t.Fatalf("anônimo em %s = %d", rota, resp.StatusCode)
		}
	}
	if resp, _ := a.getCom(t, gestor, "/api/lotes/"+id); resp.StatusCode != 200 {
		t.Fatalf("gestor no lote da Ana = %d", resp.StatusCode)
	}
	if resp, _ := a.postJSON(t, bia, "/api/lotes/"+id+"/cessao", nil); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Bia cedendo o lote da Ana = %d", resp.StatusCode)
	}
	// sem login continua tudo funcionando (curl dos jurados)
	if _, cfgAPI := a.get(t, "/api/config"); cfgAPI["exigir_login"] != false {
		t.Fatalf("/api/config deveria avisar exigir_login=false: %v", cfgAPI["exigir_login"])
	}
	_, anon := a.enviar(t, gerar(t, 30, 77), "")
	if l := a.esperar(t, anon["lote_id"].(string)); l["status"] != "concluido" {
		t.Fatalf("lote anônimo = %v", l)
	}
}

func TestExigirLogin(t *testing.T) {
	amb := ambiente(t)
	resetar(t, amb.DatabaseURL)
	cfg, _ := os.ReadFile(raiz + "config/antecipe.yaml")
	caminho := t.TempDir() + "/antecipe.yaml"
	ligado := strings.Replace(string(cfg), "exigir_login: false", "exigir_login: true", 1)
	if ligado == string(cfg) {
		t.Fatal("config/antecipe.yaml deveria trazer exigir_login: false")
	}
	_ = os.WriteFile(caminho, []byte(ligado), 0o644)
	amb.ConfigPath = caminho
	a := iniciar(t, amb, nil)
	if _, cfgAPI := a.get(t, "/api/config"); cfgAPI["exigir_login"] != true {
		t.Fatalf("/api/config deveria avisar exigir_login=true: %v", cfgAPI["exigir_login"])
	}
	resp, r := a.enviar(t, exemplo(t), "")
	if resp.StatusCode != http.StatusUnauthorized || r["erro"].(map[string]any)["codigo"] != "NAO_AUTENTICADO" {
		t.Fatalf("envio sem login com exigir_login = %d %v", resp.StatusCode, r)
	}
	c := navegador(t)
	a.postJSON(t, c, "/api/auth/cadastro", cadastro("ana@x.com"))
	if r := a.enviarCom(t, c, exemplo(t)); r["lote_id"] == nil {
		t.Fatalf("envio com login = %v", r)
	}
}

func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }

// Uma GESTOR_SENHA longa demais para o bcrypt (> 72 bytes) não pode derrubar a subida:
// a API sobe, avisa no log e a senha continua valendo pelo header X-Gestor-Senha.
func TestGestorSenhaLongaNaoDerrubaASubida(t *testing.T) {
	amb := ambiente(t)
	resetar(t, amb.DatabaseURL)
	amb.GestorSenha = strings.Repeat("s", 80)
	var logs bytes.Buffer
	a := iniciar(t, amb, &logs)
	if resp, _ := a.get(t, "/api/saude"); resp.StatusCode != http.StatusOK {
		t.Fatalf("saude = %d", resp.StatusCode)
	}
	req, _ := http.NewRequest(http.MethodGet, a.url+"/api/gestor/verificar", nil)
	req.Header.Set("X-Gestor-Senha", amb.GestorSenha)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("verificar com header = %d", resp.StatusCode)
	}
	if !strings.Contains(logs.String(), "gestor inicial") || strings.Contains(logs.String(), amb.GestorSenha) {
		t.Fatalf("esperava aviso sem a senha no log: %s", logs.String())
	}
}
