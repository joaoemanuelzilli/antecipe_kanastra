// Package postgres é o adaptador de saída para o PostgreSQL (pgx). Implementa as portas do domínio.
package postgres

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migracoes embed.FS

const (
	travaMigracao = 424242
	travaSemente  = 424243
)

// Repo implementa as portas de persistência.
type Repo struct {
	Pool *pgxpool.Pool
}

// Abrir conecta e confere a conexão.
func Abrir(ctx context.Context, url string) (*Repo, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("postgres: configuração inválida")
	}
	cfg.MaxConns = 16
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: banco indisponível: %w", err)
	}
	return &Repo{Pool: pool}, nil
}

// Fechar encerra o pool.
func (r *Repo) Fechar() { r.Pool.Close() }

// Ping confere se o banco responde (usado em /api/saude).
func (r *Repo) Ping(ctx context.Context) error { return r.Pool.Ping(ctx) }

// Migrar aplica os arquivos .sql embutidos, em ordem, sob uma trava (várias instâncias podem subir juntas).
func (r *Repo) Migrar(ctx context.Context) error {
	conn, err := r.Pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, "SELECT pg_advisory_lock($1)", travaMigracao); err != nil {
		return err
	}
	defer conn.Exec(context.Background(), "SELECT pg_advisory_unlock($1)", travaMigracao) //nolint:errcheck

	nomes, err := fs.Glob(migracoes, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(nomes)
	for _, n := range nomes {
		sql, err := migracoes.ReadFile(n)
		if err != nil {
			return err
		}
		if _, err := conn.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("migração %s: %w", n, err)
		}
	}
	return nil
}
