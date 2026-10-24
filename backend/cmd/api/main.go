// Comando api: servidor HTTP do Antecipe. Só liga as peças; a lógica mora em internal/.
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

	"antecipe/internal/platform/observability"
)

func main() {
	log := observability.NovoLogger(os.Stdout, env("LOG_LEVEL", "info"))
	if err := rodar(log); err != nil {
		log.Error("api encerrada com erro", slog.String("erro", err.Error()))
		os.Exit(1)
	}
}

func rodar(log *slog.Logger) error {
	ctx, parar := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer parar()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/saude", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	srv := &http.Server{
		Addr:              env("HTTP_ADDR", ":8080"),
		Handler:           observability.Middleware(log, mux),
		ReadHeaderTimeout: 10 * time.Second,
	}

	erros := make(chan error, 1)
	go func() {
		log.Info("api ouvindo", slog.String("endereco", srv.Addr))
		erros <- srv.ListenAndServe()
	}()

	select {
	case err := <-erros:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		log.Info("encerrando api")
		desligar, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancelar()
		return srv.Shutdown(desligar)
	}
	return nil
}

func env(chave, padrao string) string {
	if v := os.Getenv(chave); v != "" {
		return v
	}
	return padrao
}
