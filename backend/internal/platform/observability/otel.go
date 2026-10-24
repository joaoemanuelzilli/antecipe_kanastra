package observability

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otelprom "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const nomeInstrumentacao = "antecipe"

// Config do OpenTelemetry. Os traces só são exportados quando OTEL_EXPORTER_OTLP_ENDPOINT
// estiver definido (no compose aponta para o Jaeger); as métricas sempre ficam em /metrics.
type Config struct {
	ServiceName string
	Version     string
}

// Iniciar configura traces (OTLP/HTTP) e métricas (formato Prometheus) e devolve
// a função de desligamento e o handler de /metrics.
func Iniciar(ctx context.Context, c Config) (func(context.Context) error, http.Handler, error) {
	res := resource.NewSchemaless(
		attribute.String("service.name", c.ServiceName),
		attribute.String("service.version", c.Version),
	)

	opcoes := []sdktrace.TracerProviderOption{sdktrace.WithResource(res)}
	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" || os.Getenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT") != "" {
		exp, err := otlptracehttp.New(ctx) // lê endpoint e headers das variáveis OTEL_* padrão
		if err != nil {
			return nil, nil, fmt.Errorf("observabilidade: exportador de traces: %w", err)
		}
		opcoes = append(opcoes, sdktrace.WithBatcher(exp))
	}
	tp := sdktrace.NewTracerProvider(opcoes...)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	reg := prometheus.NewRegistry()
	exp, err := otelprom.New(otelprom.WithRegisterer(reg))
	if err != nil {
		return nil, nil, fmt.Errorf("observabilidade: exportador de métricas: %w", err)
	}
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exp), sdkmetric.WithResource(res))
	otel.SetMeterProvider(mp)

	desligar := func(ctx context.Context) error {
		return errors.Join(tp.Shutdown(ctx), mp.Shutdown(ctx))
	}
	return desligar, promhttp.HandlerFor(reg, promhttp.HandlerOpts{}), nil
}

// EncaminharErros manda os erros internos do OpenTelemetry (ex.: exportador sem conexão) para o log JSON.
func EncaminharErros(log *slog.Logger) {
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		log.Warn("erro do opentelemetry", slog.String("erro", err.Error()))
	}))
}

// Tracer devolve o tracer da aplicação (usa o provider global).
func Tracer() trace.Tracer { return otel.Tracer(nomeInstrumentacao) }

// Meter devolve o meter da aplicação (usa o provider global).
func Meter() metric.Meter { return otel.Meter(nomeInstrumentacao) }
