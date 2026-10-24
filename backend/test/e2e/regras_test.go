package e2e

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
)

func (a *app) gestor(t *testing.T, metodo, caminho string, corpo any, senha string) (*http.Response, map[string]any) {
	t.Helper()
	var b []byte
	if corpo != nil {
		b, _ = json.Marshal(corpo)
	}
	req, _ := http.NewRequest(metodo, a.url+caminho, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if senha != "" {
		req.Header.Set("X-Gestor-Senha", senha)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp, lerJSON(t, resp)
}

// regrasCom devolve as regras publicadas com o prazo máximo trocado.
func (a *app) regrasCom(t *testing.T, prazo any) map[string]any {
	t.Helper()
	_, r := a.get(t, "/api/regras")
	regras := r["regras"].(map[string]any)
	for _, v := range regras["validadores"].([]any) {
		m := v.(map[string]any)
		if m["codigo"] == "PRAZO_MAXIMO" {
			m["parametros"].(map[string]any)["prazo_maximo_dias"] = prazo
		}
	}
	return regras
}

func TestRegrasMostraCatalogoEMetodoDoYAML(t *testing.T) {
	a := novoApp(t)
	resp, r := a.get(t, "/api/regras")
	if resp.StatusCode != 200 || num(r["versao"]) != 1 || len(r["catalogo"].([]any)) != 9 {
		t.Fatalf("regras = %d %v", resp.StatusCode, r)
	}
	p := r["precificacao_efetiva"].(map[string]any)
	sob := p["sobrescritos_pelo_yaml"].([]any)
	if p["metodo"] != "DESCONTO_COMPOSTO" || len(sob) != 1 || sob[0] != "metodo" || p["fonte_metodo"] != "config/antecipe.yaml" {
		t.Fatalf("precificação efetiva = %v", p)
	}
	if r["regras"].(map[string]any)["precificacao"].(map[string]any)["metodo"] != "DESCONTO_SIMPLES" {
		t.Fatal("o JSON das regras deve continuar como o original")
	}
}

func TestGestorSenha(t *testing.T) {
	a := novoApp(t)
	if resp, _ := a.gestor(t, http.MethodGet, "/api/gestor/verificar", nil, "senha-de-teste"); resp.StatusCode != http.StatusNoContent {
		t.Fatalf("senha certa = %d", resp.StatusCode)
	}
	resp, r := a.gestor(t, http.MethodGet, "/api/gestor/verificar", nil, "errada")
	if resp.StatusCode != http.StatusUnauthorized || r["erro"].(map[string]any)["codigo"] != "SENHA_INVALIDA" {
		t.Fatalf("senha errada = %d %v", resp.StatusCode, r)
	}
	resp, _ = a.gestor(t, http.MethodPut, "/api/regras", map[string]any{"regras": a.regrasCom(t, 120), "autor": "Rafael"}, "")
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("PUT sem senha = %d", resp.StatusCode)
	}
}

func TestPutRegrasCriaVersaoENovoLoteUsa(t *testing.T) {
	a := novoApp(t)
	_, r1 := a.enviar(t, exemplo(t), "")
	l1 := a.esperar(t, r1["lote_id"].(string))

	resp, p := a.gestor(t, http.MethodPut, "/api/regras", map[string]any{"regras": a.regrasCom(t, 30), "autor": "Rafael"}, "senha-de-teste")
	if resp.StatusCode != http.StatusCreated || num(p["versao"]) != 2 {
		t.Fatalf("PUT = %d %v", resp.StatusCode, p)
	}
	_, v := a.get(t, "/api/regras/versoes")
	versoes := v["versoes"].([]any)
	resumo := versoes[0].(map[string]any)["resumo"].([]any)
	if len(versoes) != 2 || len(resumo) != 1 || resumo[0] != "PRAZO_MAXIMO: 180 → 30 dias" || versoes[0].(map[string]any)["autor"] != "Rafael" {
		t.Fatalf("versões = %v", versoes)
	}
	_, vj := a.get(t, "/api/regras/versoes/1")
	if num(vj["versao"]) != 1 || vj["regras"] == nil {
		t.Fatalf("versão 1 = %v", vj)
	}
	// sem deploy: reenviar o mesmo arquivo gera um lote novo com a versão 2 e menos aprovadas
	resp, r2 := a.enviar(t, exemplo(t), "")
	if resp.StatusCode != http.StatusAccepted || r2["lote_id"] == r1["lote_id"] {
		t.Fatalf("reenvio após mudar regra = %d %v", resp.StatusCode, r2)
	}
	l2 := a.esperar(t, r2["lote_id"].(string))
	if num(l2["regras_versao"]) != 2 || num(l2["totais"].(map[string]any)["aprovadas"]) >= num(l1["totais"].(map[string]any)["aprovadas"]) {
		t.Fatalf("lote novo = versão %v, aprovadas %v (antes %v)", l2["regras_versao"], l2["totais"], l1["totais"])
	}
}

func TestPutRegrasInvalidas(t *testing.T) {
	a := novoApp(t)
	resp, r := a.gestor(t, http.MethodPut, "/api/regras", map[string]any{"regras": a.regrasCom(t, -1), "autor": "Rafael"}, "senha-de-teste")
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("PUT inválido = %d %v", resp.StatusCode, r)
	}
	e := r["erro"].(map[string]any)
	det := e["detalhes"].([]any)
	if e["codigo"] != "REGRAS_INVALIDAS" || len(det) != 1 || det[0].(map[string]any)["campo"] != "validadores[5].parametros.prazo_maximo_dias" {
		t.Fatalf("erro = %v", e)
	}
	resp, r = a.gestor(t, http.MethodPut, "/api/regras", map[string]any{"regras": a.regrasCom(t, 180), "autor": "Rafael"}, "senha-de-teste")
	if resp.StatusCode != http.StatusUnprocessableEntity || r["erro"].(map[string]any)["codigo"] != "SEM_MUDANCAS" {
		t.Fatalf("PUT sem mudança = %d %v", resp.StatusCode, r)
	}
}

func TestPreviaMudaAprovadas(t *testing.T) {
	a := novoApp(t)
	resp, r := a.gestor(t, http.MethodPost, "/api/regras/previa", map[string]any{"regras": a.regrasCom(t, 30)}, "senha-de-teste")
	if resp.StatusCode != http.StatusConflict || r["erro"].(map[string]any)["codigo"] != "SEM_LOTE" {
		t.Fatalf("prévia sem lote = %d %v", resp.StatusCode, r)
	}
	_, env := a.enviar(t, exemplo(t), "")
	a.esperar(t, env["lote_id"].(string))
	resp, p := a.gestor(t, http.MethodPost, "/api/regras/previa", map[string]any{"regras": a.regrasCom(t, 30)}, "senha-de-teste")
	if resp.StatusCode != 200 {
		t.Fatalf("prévia = %d %v", resp.StatusCode, p)
	}
	antes, depois := p["antes"].(map[string]any), p["depois"].(map[string]any)
	if num(depois["aprovadas"]) >= num(antes["aprovadas"]) || num(depois["liquido_centavos"]) >= num(antes["liquido_centavos"]) {
		t.Fatalf("prévia = %v", p)
	}
	m := p["mudancas"].([]any)
	if len(m) != 1 || m[0] != "PRAZO_MAXIMO: 180 → 30 dias" || p["lote_id"] != env["lote_id"] || num(p["linhas"]) != 23 {
		t.Fatalf("prévia = %v", p)
	}
	// a prévia não publica nada
	_, reg := a.get(t, "/api/regras")
	if num(reg["versao"]) != 1 {
		t.Fatalf("prévia publicou: versão %v", reg["versao"])
	}
}
