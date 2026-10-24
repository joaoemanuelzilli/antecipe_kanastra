package domain

import (
	"context"
	"time"
)

// Perfis de usuário.
const (
	PerfilEmpresa = "empresa"
	PerfilGestor  = "gestor"
)

// Usuario é uma conta. A senha nunca sai do repositório.
type Usuario struct {
	ID          string
	Email       string
	Nome        string
	Perfil      string
	EmpresaNome string
	EmpresaCNPJ string
	CriadoEm    time.Time
}

// NovoUsuario são os dados para criar uma conta (SenhaHash já calculado pelo serviço).
type NovoUsuario struct {
	Email, Nome, Perfil, EmpresaNome, EmpresaCNPJ, SenhaHash string
}

// ResumoLote é um item da lista "meus lotes".
type ItemLote struct {
	ID          string
	Status      string
	DataBase    time.Time
	CriadoEm    time.Time
	ConcluidoEm *time.Time
	Totais      *Totais
	UsuarioID   string
	TemCessao   bool
}

// UsuarioRepository guarda contas, sessões e o vínculo dos lotes com as contas.
type UsuarioRepository interface {
	CriarUsuario(ctx context.Context, n NovoUsuario) (Usuario, error)
	GarantirGestor(ctx context.Context, n NovoUsuario) error
	DesativarGestor(ctx context.Context, email string) error
	CredenciaisPorEmail(ctx context.Context, email string) (Usuario, string, error) // usuário e hash da senha
	CriarSessao(ctx context.Context, tokenHash, usuarioID string, expira time.Time) error
	UsuarioDaSessao(ctx context.Context, tokenHash string) (Usuario, error)
	ApagarSessao(ctx context.Context, tokenHash string) error
	ListarLotes(ctx context.Context, usuarioID string, limite int) ([]ItemLote, error) // usuarioID vazio = todos
}
