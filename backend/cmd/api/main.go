// Comando api: servidor HTTP do Antecipe. Só liga as peças (via bootstrap); a lógica mora em internal/.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"antecipe/internal/bootstrap"
	"antecipe/internal/platform/observability"
)

func main() {
	amb := bootstrap.AmbienteDoSistema()
	log := observability.NovoLogger(os.Stdout, amb.LogLevel)
	if err := rodar(amb, log); err != nil {
		log.Error("api encerrada com erro", slog.String("erro", err.Error()))
		os.Exit(1)
	}
}

func rodar(amb bootstrap.Ambiente, log *slog.Logger) error {
	ctx, parar := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer parar()

	nome := os.Getenv("OTEL_SERVICE_NAME")
	if nome == "" {
		nome = "antecipe-api"
	}
	observability.EncaminharErros(log)
	desligarOtel, metricas, err := observability.Iniciar(ctx, observability.Config{ServiceName: nome, Version: "1.0.0"})
	if err != nil {
		return err
	}
	defer desligarOtel(context.Background()) //nolint:errcheck
	amb.Metricas = metricas

	app, err := bootstrap.Novo(ctx, amb, log)
	if err != nil {
		return err
	}
	srv := &http.Server{Addr: amb.HTTPAddr, Handler: app.Handler, ReadHeaderTimeout: 10 * time.Second}

	erros := make(chan error, 1)
	go func() {
		log.Info("api ouvindo", slog.String("endereco", srv.Addr))
		erros <- srv.ListenAndServe()
	}()
	select {
	case err := <-erros:
		app.Fechar(context.Background())
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		log.Info("encerrando api")
		desligar, cancelar := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancelar()
		_ = srv.Shutdown(desligar)
		app.Fechar(desligar)
	}
	return nil
}
