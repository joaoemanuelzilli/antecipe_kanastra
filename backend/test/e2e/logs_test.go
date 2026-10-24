package e2e

import (
	"bytes"
	"net/http"
	"strings"
	"sync"
	"testing"

	"antecipe/internal/handler/csvfile"
	"antecipe/internal/platform/document"
)

type bufSeguro struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *bufSeguro) Write(p []byte) (int, error) { s.mu.Lock(); defer s.mu.Unlock(); return s.b.Write(p) }
func (s *bufSeguro) String() string            { s.mu.Lock(); defer s.mu.Unlock(); return s.b.String() }

// TestLogsNaoVazamDados passa pelos fluxos principais e confere que nenhum dado do CSV
// (nomes, documentos, números de nota) nem segredo aparece nos logs.
func TestLogsNaoVazamDados(t *testing.T) {
	amb := ambiente(t)
	resetar(t, amb.DatabaseURL)
	logs := &bufSeguro{}
	a := iniciar(t, amb, logs)

	conteudo := exemplo(t)
	_, r := a.enviar(t, conteudo, "")
	id := r["lote_id"].(string)
	a.esperar(t, id)
	a.get(t, "/api/lotes/"+id+"/notas?busca=AURORA")
	resp, err := http.Post(a.url+"/api/validar", "text/csv", bytes.NewReader(conteudo))
	if err != nil {
		t.Fatal(err)
	}
	lerJSON(t, resp)
	a.ceder(t, id, "")
	a.gestor(t, http.MethodPut, "/api/regras", map[string]any{"regras": a.regrasCom(t, 120), "autor": "Rafael"}, "senha-de-teste")
	a.gestor(t, http.MethodGet, "/api/gestor/verificar", nil, "senha-errada-123")
	a.enviar(t, []byte("a;b\n1;2\n"), "")

	texto := logs.String()
	if !strings.Contains(texto, `"msg":"lote concluido"`) {
		t.Fatalf("logs não capturados: %s", texto)
	}
	notas, err := csvfile.Ler(bytes.NewReader(conteudo), nil)
	if err != nil {
		t.Fatal(err)
	}
	proibidos := []string{"senha-de-teste", "senha-errada-123", "postgres://", amb.DatabaseURL, "AURORA"}
	for _, n := range notas {
		for _, v := range []string{n.DocSacado, n.DocCedente, document.Normalizar(n.DocSacado), document.Normalizar(n.DocCedente), n.NomeSacado, n.NomeCedente} {
			if len(v) >= 5 {
				proibidos = append(proibidos, v)
			}
		}
		if len(n.Numero) >= 5 {
			proibidos = append(proibidos, n.Numero)
		}
	}
	for _, p := range proibidos {
		if strings.Contains(texto, p) {
			t.Errorf("log vazou %q", p)
		}
	}
}
