package postgres

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"antecipe/internal/domain/rules"
	"antecipe/internal/platform/document"
	"antecipe/internal/repository/filesystem"
)

// novoRepo conecta no banco de teste (TEST_DATABASE_URL), zera o schema e aplica as migrations.
func novoRepo(t *testing.T) *Repo {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL não definida")
	}
	ctx := context.Background()
	r, err := Abrir(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(r.Fechar)
	if _, err := r.Pool.Exec(ctx, "DROP SCHEMA public CASCADE; CREATE SCHEMA public"); err != nil {
		t.Fatal(err)
	}
	if err := r.Migrar(ctx); err != nil {
		t.Fatal(err)
	}
	return r
}

func semente(t *testing.T) func() (Semente, error) {
	t.Helper()
	return func() (Semente, error) {
		est, err := filesystem.LerEstoqueZip("../../../../data/estoque_antecipe.zip")
		if err != nil {
			return Semente{}, err
		}
		b, err := os.ReadFile("../../../../data/validadores_antecipe.json")
		if err != nil {
			return Semente{}, err
		}
		reg, err := rules.Ler(b)
		if err != nil {
			return Semente{}, err
		}
		sit, err := filesystem.LerSituacoes("../../../../data/situacao_cadastral.csv")
		return Semente{Estoque: est, Regras: reg, Situacoes: sit}, err
	}
}

func contar(t *testing.T, r *Repo, tabela string) int {
	t.Helper()
	var n int
	if err := r.Pool.QueryRow(context.Background(), "SELECT count(*) FROM "+tabela).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestMigrarIdempotente(t *testing.T) {
	r := novoRepo(t)
	if err := r.Migrar(context.Background()); err != nil {
		t.Fatalf("segunda migração: %v", err)
	}
}

func TestSemente(t *testing.T) {
	r := novoRepo(t)
	ctx := context.Background()
	if err := r.SemearSeVazio(ctx, semente(t)); err != nil {
		t.Fatal(err)
	}
	if n := contar(t, r, "estoque"); n != 100554 {
		t.Fatalf("estoque = %d", n)
	}
	if n := contar(t, r, "situacao_cadastral"); n != 6 {
		t.Fatalf("situações = %d", n)
	}
	v, err := r.Publicadas(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if v.Numero != 1 || len(v.Regras.Validadores) != 9 || v.Autor != "semente" || len(v.Resumo) != 1 {
		t.Fatalf("versão = %+v", v)
	}
	chamou := false
	if err := r.SemearSeVazio(ctx, func() (Semente, error) { chamou = true; return Semente{}, nil }); err != nil {
		t.Fatal(err)
	}
	if chamou {
		t.Fatal("semente carregada de novo com o banco já populado")
	}
	if n := contar(t, r, "estoque"); n != 100554 {
		t.Fatalf("estoque depois da segunda chamada = %d", n)
	}
}

func TestContextoMotor(t *testing.T) {
	r := novoRepo(t)
	ctx := context.Background()
	if err := r.SemearSeVazio(ctx, semente(t)); err != nil {
		t.Fatal(err)
	}
	vendidas, est, err := r.ContextoMotor(ctx, "92.234.400/0001-80", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(vendidas) != 100554 || est.TotalCentavos != 5307402800 {
		t.Fatalf("vendidas = %d, total = %d", len(vendidas), est.TotalCentavos)
	}
	var raiz int64
	for doc, v := range est.PorSacado {
		if document.Raiz(doc) == "58206002" {
			raiz += v
		}
	}
	if pct := float64(raiz) * 100 / float64(est.TotalCentavos); pct < 12.49 || pct > 12.51 {
		t.Fatalf("concentração = %.3f", pct)
	}
	sit, err := r.Situacoes(ctx)
	if err != nil || sit["11122233000183"] != "INAPTA" {
		t.Fatalf("situações = %v, %v", sit, err)
	}
}

// TestAbrirEsperaOBanco simula o Postgres que ainda não aceita TCP na subida do compose:
// a porta só passa a responder depois de 1,5 s, e Abrir precisa esperar em vez de falhar.
func TestAbrirEsperaOBanco(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL não definida")
	}
	destino, err := pgconnHostPort(url)
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	porta := ln.Addr().String()
	ln.Close() // nada escutando por enquanto
	go func() {
		time.Sleep(1500 * time.Millisecond)
		l, err := net.Listen("tcp", porta)
		if err != nil {
			return
		}
		defer l.Close()
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go encaminhar(c, destino)
		}
	}()
	urlAtrasada := strings.Replace(url, destino, porta, 1)
	r, err := AbrirComEspera(context.Background(), urlAtrasada, 20, 200*time.Millisecond)
	if err != nil {
		t.Fatalf("Abrir desistiu cedo: %v", err)
	}
	r.Fechar()
}

func pgconnHostPort(url string) (string, error) {
	i := strings.Index(url, "@")
	j := strings.Index(url[i:], "/")
	if i < 0 || j < 0 {
		return "", errors.New("url sem host")
	}
	return url[i+1 : i+j], nil
}

func encaminhar(c net.Conn, destino string) {
	d, err := net.Dial("tcp", destino)
	if err != nil {
		c.Close()
		return
	}
	go func() { io.Copy(d, c); d.Close() }()
	io.Copy(c, d)
	c.Close()
}
