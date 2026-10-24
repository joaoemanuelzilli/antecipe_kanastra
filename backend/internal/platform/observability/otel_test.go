package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestLoggerComTraceID(t *testing.T) {
	var buf bytes.Buffer
	log := NovoLogger(&buf, "info")
	tp := sdktrace.NewTracerProvider()
	ctx, span := tp.Tracer("teste").Start(context.Background(), "op")
	log.InfoContext(ctx, "oi")
	span.End()

	var linha map[string]any
	if err := json.Unmarshal(buf.Bytes(), &linha); err != nil {
		t.Fatal(err)
	}
	if linha["trace_id"] != span.SpanContext().TraceID().String() || linha["span_id"] == nil {
		t.Fatalf("log sem trace_id: %v", linha)
	}
}

func TestMiddlewareCriaSpanComRota(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	anterior := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(anterior)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/lotes/{id}", func(w http.ResponseWriter, r *http.Request) {})
	h := Middleware(NovoLogger(io.Discard, "info"), mux)
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/lotes/abc", nil))

	spans := rec.Ended()
	if len(spans) != 1 || spans[0].Name() != "GET /api/lotes/{id}" {
		t.Fatalf("spans = %v", nomes(spans))
	}
	achou := false
	for _, a := range spans[0].Attributes() {
		if a.Key == attribute.Key("http.route") && a.Value.AsString() == "/api/lotes/{id}" {
			achou = true
		}
	}
	if !achou {
		t.Fatalf("atributo http.route ausente: %v", spans[0].Attributes())
	}
}

func TestMiddlewareNaoRastreiaSaude(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	anterior := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(anterior)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/saude", func(w http.ResponseWriter, r *http.Request) {})
	Middleware(NovoLogger(io.Discard, "info"), mux).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/saude", nil))
	if len(rec.Ended()) != 0 {
		t.Fatalf("saúde não deveria gerar span: %v", nomes(rec.Ended()))
	}
}

func TestIniciarExpoeMetricas(t *testing.T) {
	desligar, metricas, err := Iniciar(context.Background(), Config{ServiceName: "antecipe-teste"})
	if err != nil {
		t.Fatal(err)
	}
	defer desligar(context.Background())

	c, err := Meter().Int64Counter("antecipe_teste_total")
	if err != nil {
		t.Fatal(err)
	}
	c.Add(context.Background(), 3)

	rec := httptest.NewRecorder()
	metricas.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if !strings.Contains(rec.Body.String(), "antecipe_teste_total") {
		t.Fatalf("métrica ausente em /metrics: %s", rec.Body.String())
	}
}

func nomes(s []sdktrace.ReadOnlySpan) []string {
	var out []string
	for _, x := range s {
		out = append(out, x.Name())
	}
	return out
}

func TestErrosDoOtelViramLogJSON(t *testing.T) {
	var buf bytes.Buffer
	EncaminharErros(NovoLogger(&buf, "info"))
	otel.Handle(errors.New("traces export: connection refused"))
	var linha map[string]any
	if err := json.Unmarshal(buf.Bytes(), &linha); err != nil || linha["msg"] != "erro do opentelemetry" {
		t.Fatalf("log = %q (%v)", buf.String(), err)
	}
}
