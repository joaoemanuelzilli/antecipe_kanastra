package domain

import (
	"context"
	"time"

	"antecipe/internal/domain/rules"
)

// Portas (interfaces) que os serviços usam. Os adaptadores em internal/repository as implementam.

// VersaoRegras é uma versão publicada das regras do fundo.
type VersaoRegras struct {
	Numero      int
	Regras      rules.Regras
	Autor       string
	PublicadaEm time.Time
	Resumo      []string
}

// EstoqueRepository dá ao motor o que já foi vendido e as somas do fundo.
type EstoqueRepository interface {
	// ignorarCessao (opcional) deixa de fora as notas de uma cessão: a prévia de um lote já cedido
	// não pode contar as notas dele mesmo como duplicadas.
	ContextoMotor(ctx context.Context, ignorarCessao string) (ContextoEstoque, error)
}

// SituacaoRepository consulta a situação cadastral (fonte simulada).
type SituacaoRepository interface {
	Situacoes(ctx context.Context) (map[string]string, error)
}

// RegrasRepository guarda as versões das regras.
type RegrasRepository interface {
	Publicadas(ctx context.Context) (VersaoRegras, error)
	Obter(ctx context.Context, numero int) (VersaoRegras, error)
	Publicar(ctx context.Context, r rules.Regras, autor string, resumo []string) (VersaoRegras, error)
	Listar(ctx context.Context) ([]VersaoRegras, error)
}
