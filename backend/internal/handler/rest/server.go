// Package rest é o adaptador de entrada HTTP: traduz requisições para os serviços e
// as respostas para o contrato de docs/api.md. Não tem regra de negócio.
package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slices"

	"antecipe/internal/platform/apperrors"
	"antecipe/internal/platform/observability"
	"antecipe/internal/service/assignment"
	"antecipe/internal/service/batch"
)

// Deps do servidor HTTP.
type Deps struct {
	Lotes       *batch.Servico
	Cessoes     *assignment.Servico
	Saude       func(ctx context.Context) error
	Metricas    http.Handler
	Log         *slog.Logger
	GestorSenha string
	CORSOrigens []string
	TmpDir      string
}

// Handlers agrupa os handlers e o que eles compartilham.
type Handlers struct {
	d Deps
}

// NovoServidor monta as rotas com os middlewares (observabilidade, CORS e recuperação de panic).
func NovoServidor(d Deps) http.Handler {
	h := &Handlers{d: d}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/saude", h.saude)
	mux.HandleFunc("GET /api/config", h.config)
	mux.HandleFunc("POST /api/lotes", h.enviarLote)
	mux.HandleFunc("GET /api/lotes/{id}", h.obterLote)
	mux.HandleFunc("GET /api/lotes/{id}/notas", h.listarNotas)
	mux.HandleFunc("GET /api/lotes/{id}/notas.csv", h.baixarNotas)
	mux.HandleFunc("POST /api/validar", h.validar)
	mux.HandleFunc("POST /api/lotes/{id}/cessao", h.ceder)
	if d.Metricas != nil {
		mux.Handle("GET /metrics", d.Metricas)
	}
	return observability.Middleware(d.Log, cors(d.CORSOrigens, recuperar(d.Log, mux)))
}

func (h *Handlers) saude(w http.ResponseWriter, r *http.Request) {
	if h.d.Saude != nil {
		if err := h.d.Saude(r.Context()); err != nil {
			EscreverErro(w, r, h.d.Log, apperrors.Unavailable("BANCO_INDISPONIVEL", "O banco de dados não está respondendo.").Wrap(err))
			return
		}
	}
	EscreverJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// EscreverJSON responde com JSON.
func EscreverJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type corpoErro struct {
	Erro struct {
		Codigo   string `json:"codigo"`
		Mensagem string `json:"mensagem"`
		Detalhes any    `json:"detalhes"`
	} `json:"erro"`
}

// EscreverErro traduz qualquer erro para o formato do contrato. Erros internos são logados
// só com o tipo (o texto pode carregar dados) e respondidos com mensagem genérica.
func EscreverErro(w http.ResponseWriter, r *http.Request, log *slog.Logger, err error) {
	ae := apperrors.From(err)
	if ae.Status >= 500 {
		log.ErrorContext(r.Context(), "erro interno", slog.String("codigo", ae.Code), slog.String("erro", fmt.Sprintf("%T", ae.Err)))
	}
	var c corpoErro
	c.Erro.Codigo, c.Erro.Mensagem, c.Erro.Detalhes = ae.Code, ae.Message, ae.Details
	if c.Erro.Detalhes == nil {
		c.Erro.Detalhes = []any{}
	}
	EscreverJSON(w, ae.Status, c)
}

func recuperar(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				EscreverErro(w, r, log, apperrors.Internal(fmt.Errorf("panic: %T", v)))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func cors(origens []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if o := r.Header.Get("Origin"); o != "" && slices.Contains(origens, o) {
			w.Header().Set("Access-Control-Allow-Origin", o)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Idempotency-Key, X-Gestor-Senha, X-Request-Id")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, OPTIONS")
			w.Header().Set("Access-Control-Expose-Headers", "Location, Idempotent-Replay, X-Request-Id")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}
