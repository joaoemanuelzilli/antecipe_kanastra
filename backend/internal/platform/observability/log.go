// Package obs configura os logs estruturados (slog JSON) e o log de acesso HTTP.
//
// Política: os logs só carregam metadados (rota, status, contagens, durações, ids).
// Nenhum campo do CSV, nome de arquivo, corpo de requisição ou segredo é logado.
package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
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
	return slog.New(comTrace{slog.NewJSONHandler(w, &slog.HandlerOptions{Level: lv})})
}

// comTrace acrescenta trace_id e span_id do OpenTelemetry a cada linha logada com contexto.
type comTrace struct{ slog.Handler }

func (h comTrace) Handle(ctx context.Context, r slog.Record) error {
	if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
		r.AddAttrs(slog.String("trace_id", sc.TraceID().String()), slog.String("span_id", sc.SpanID().String()))
	}
	return h.Handler.Handle(ctx, r)
}

func (h comTrace) WithAttrs(a []slog.Attr) slog.Handler { return comTrace{h.Handler.WithAttrs(a)} }
func (h comTrace) WithGroup(n string) slog.Handler      { return comTrace{h.Handler.WithGroup(n)} }

// RequestID devolve o id da requisição guardado no contexto, ou "".
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(chaveCtx{}).(string)
	return id
}

// ComRequestID guarda o id da requisição no contexto.
func ComRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, chaveCtx{}, id)
}

// Middleware cria um span OpenTelemetry por requisição (nomeado pela rota), gera ou propaga
// o X-Request-Id e registra uma linha de acesso. /api/saude e /metrics não geram spans.
func Middleware(log *slog.Logger, next http.Handler) http.Handler {
	acesso := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
		span := trace.SpanFromContext(r.Context())
		span.SetName(rota)
		if _, caminho, ok := strings.Cut(rota, " "); ok {
			span.SetAttributes(attribute.String("http.route", caminho))
		}
		span.SetAttributes(attribute.String("request_id", id))
		log.InfoContext(r.Context(), "requisicao",
			slog.String("request_id", id),
			slog.String("metodo", r.Method),
			slog.String("rota", rota),
			slog.Int("status", rw.status),
			slog.Int64("duracao_ms", time.Since(inicio).Milliseconds()),
		)
	})
	return otelhttp.NewHandler(acesso, "http",
		otelhttp.WithFilter(func(r *http.Request) bool { return r.URL.Path != "/api/saude" && r.URL.Path != "/metrics" }))
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
