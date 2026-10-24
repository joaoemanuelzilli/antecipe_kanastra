package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

func (a *app) validarBruto(t *testing.T, tipo, corpo string) (*http.Response, map[string]any) {
	t.Helper()
	resp, err := http.Post(a.url+"/api/validar", tipo, strings.NewReader(corpo))
	if err != nil {
		t.Fatal(err)
	}
	return resp, lerJSON(t, resp)
}

func TestValidarErrosDeArquivo(t *testing.T) {
	a := novoApp(t)
	resp, r := a.validarBruto(t, "text/csv", "NOME_FUNDO,DOC_FUNDO,NOME_CEDENTE,DOC_CEDENTE,NOME_SACADO,DOC_SACADO,NU_DOCUMENTO,TIPO_RECEBIVEL,VALOR_NOMINAL,DATA_VENCIMENTO_ORIGINAL\n1,2,3,4,5,6,7,8,9,10\n")
	e, _ := r["erro"].(map[string]any)
	if resp.StatusCode != http.StatusUnprocessableEntity || e["codigo"] != "CABECALHO_INVALIDO" || !strings.Contains(e["mensagem"].(string), "ponto e vírgula") {
		t.Fatalf("csv com vírgula = %d %v", resp.StatusCode, r)
	}
	resp, r = a.validarBruto(t, "application/json", `{"notas":[]}`)
	if resp.StatusCode != http.StatusUnprocessableEntity || r["erro"].(map[string]any)["codigo"] != "ARQUIVO_VAZIO" {
		t.Fatalf("sem notas = %d %v", resp.StatusCode, r)
	}
}

func TestValidarJSONComAspasEPontoEVirgula(t *testing.T) {
	a := novoApp(t)
	nota := func(sacado, numero string) map[string]string {
		return map[string]string{"nome_fundo": "ANTECIPE FIDC", "doc_fundo": "92.234.400/0001-80", "nome_cedente": "CED", "doc_cedente": "10.062.050/0001-57",
			"nome_sacado": sacado, "doc_sacado": "55.566.677/0001-83", "nu_documento": numero, "tipo_recebivel": "Duplicata",
			"valor_nominal": "1000,00", "data_vencimento_original": "21/11/2026"}
	}
	b, _ := json.Marshal(map[string]any{"notas": []any{nota(`"ACME; COMÉRCIO`, "A1"), nota("LOJA NORMAL", "A2")}})
	resp, r := a.validarBruto(t, "application/json", string(b))
	if resp.StatusCode != 200 {
		t.Fatalf("validar = %d %v", resp.StatusCode, r)
	}
	notas := r["notas"].([]any)
	if len(notas) != 2 || notas[0].(map[string]any)["sacado"].(map[string]any)["nome"] != `"ACME; COMÉRCIO` || notas[1].(map[string]any)["status"] != "aprovada" {
		t.Fatalf("notas = %v", notas)
	}
}

func TestNotasCSVComBOM(t *testing.T) {
	a := novoApp(t)
	_, r := a.enviar(t, exemplo(t), "")
	id := r["lote_id"].(string)
	a.esperar(t, id)
	resp, err := http.Get(a.url + "/api/lotes/" + id + "/notas.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if !bytes.HasPrefix(b, []byte("\xEF\xBB\xBFNOME_FUNDO;")) {
		t.Fatalf("CSV sem BOM: %q", b[:20])
	}
}

func TestPreviaDepoisDaCessao(t *testing.T) {
	a := novoApp(t)
	_, r := a.enviar(t, exemplo(t), "")
	id := r["lote_id"].(string)
	a.esperar(t, id)
	if resp, c := a.ceder(t, id, ""); resp.StatusCode != http.StatusCreated {
		t.Fatalf("cessão = %d %v", resp.StatusCode, c)
	}
	// prazo maior é mais permissivo: a prévia não pode mostrar menos aprovadas por "duplicidade" com a própria cessão
	resp, p := a.gestor(t, http.MethodPost, "/api/regras/previa", map[string]any{"regras": a.regrasCom(t, 240), "lote_id": id}, "senha-de-teste")
	if resp.StatusCode != 200 {
		t.Fatalf("prévia = %d %v", resp.StatusCode, p)
	}
	antes, depois := num(p["antes"].(map[string]any)["aprovadas"]), num(p["depois"].(map[string]any)["aprovadas"])
	if depois < antes {
		t.Fatalf("prévia após a cessão: aprovadas %v → %v (%v)", antes, depois, p["motivos"])
	}
}
