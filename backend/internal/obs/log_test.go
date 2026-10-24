package obs

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNovoLoggerJSON(t *testing.T) {
	var buf bytes.Buffer
	log := NovoLogger(&buf, "info")
	log.Info("oi")

	var linha map[string]any
	if err := json.Unmarshal(buf.Bytes(), &linha); err != nil {
		t.Fatalf("log não é JSON: %v (%q)", err, buf.String())
	}
	if linha["level"] != "INFO" || linha["msg"] != "oi" {
		t.Fatalf("campos inesperados: %v", linha)
	}
}

func TestNovoLoggerNivel(t *testing.T) {
	var buf bytes.Buffer
	log := NovoLogger(&buf, "warn")
	log.Info("não deve aparecer")
	if buf.Len() != 0 {
		t.Fatalf("info logado com nível warn: %q", buf.String())
	}
}

func TestMiddlewareCampos(t *testing.T) {
	var buf bytes.Buffer
	log := NovoLogger(&buf, "info")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/saude", func(w http.ResponseWriter, r *http.Request) {
		if RequestID(r.Context()) == "" {
			t.Error("request id ausente no contexto")
		}
		w.WriteHeader(http.StatusOK)
	})
	h := Middleware(log, mux)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/saude", nil))

	id := rec.Header().Get("X-Request-Id")
	if id == "" {
		t.Fatal("header X-Request-Id ausente")
	}
	var linha map[string]any
	if err := json.Unmarshal(buf.Bytes(), &linha); err != nil {
		t.Fatalf("log não é JSON: %v (%q)", err, buf.String())
	}
	if linha["rota"] != "GET /api/saude" {
		t.Errorf("rota = %v", linha["rota"])
	}
	if linha["status"] != float64(200) {
		t.Errorf("status = %v", linha["status"])
	}
	if linha["request_id"] != id {
		t.Errorf("request_id = %v, header = %s", linha["request_id"], id)
	}
	if _, ok := linha["duracao_ms"]; !ok {
		t.Error("duracao_ms ausente")
	}
}

func TestMiddlewarePropagaRequestID(t *testing.T) {
	var buf bytes.Buffer
	h := Middleware(NovoLogger(&buf, "info"), http.NotFoundHandler())
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("X-Request-Id", "abc-123")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("X-Request-Id") != "abc-123" {
		t.Fatalf("request id não propagado: %q", rec.Header().Get("X-Request-Id"))
	}
}
