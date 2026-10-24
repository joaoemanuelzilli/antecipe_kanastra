// Package obs configura os logs estruturados (slog JSON) e o log de acesso HTTP.
//
// Política: os logs só carregam metadados (rota, status, contagens, durações, ids).
// Nenhum campo do CSV, nome de arquivo, corpo de requisição ou segredo é logado.
package obs

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type chaveCtx struct{}

// NovoLogger cria um logger JSON no nível pedido (debug, info, warn, error).
func NovoLogger(w io.Writer, nivel string) *slog.Logger {
	var lv slog.Level
	switch strings.ToLower(strings.TrimSpace(nivel)) {
	case "debug":
		lv = slog.LevelDebug
	case "warn", "warning":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: lv}))
}

// RequestID devolve o id da requisição guardado no contexto, ou "".
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(chaveCtx{}).(string)
	return id
}

// ComRequestID guarda o id da requisição no contexto.
func ComRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, chaveCtx{}, id)
}

// Middleware gera (ou propaga) o X-Request-Id e registra uma linha de acesso por requisição.
func Middleware(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inicio := time.Now()
		id := r.Header.Get("X-Request-Id")
		if id == "" || len(id) > 64 {
			id = novoID()
		}
		w.Header().Set("X-Request-Id", id)
		rw := &respostaGravada{ResponseWriter: w, status: http.StatusOK}
		r = r.WithContext(ComRequestID(r.Context(), id))

		next.ServeHTTP(rw, r)

		rota := r.Pattern
		if rota == "" {
			rota = "sem_rota"
		}
		log.Info("requisicao",
			slog.String("request_id", id),
			slog.String("metodo", r.Method),
			slog.String("rota", rota),
			slog.Int("status", rw.status),
			slog.Int64("duracao_ms", time.Since(inicio).Milliseconds()),
		)
	})
}

type respostaGravada struct {
	http.ResponseWriter
	status int
}

func (r *respostaGravada) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Flush mantém o suporte a streaming (CSV de download) através do middleware.
func (r *respostaGravada) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (r *respostaGravada) Unwrap() http.ResponseWriter { return r.ResponseWriter }

func novoID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
