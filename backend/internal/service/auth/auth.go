// Package auth é o caso de uso das contas: cadastro de empresa, login, sessão e gestores.
// Senhas com bcrypt; o token da sessão é aleatório e só o hash dele vai para o banco.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"log/slog"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"antecipe/internal/domain"
	"antecipe/internal/platform/apperrors"
	"antecipe/internal/platform/document"
)

// DuracaoSessao é o tempo de vida de uma sessão.
const DuracaoSessao = 7 * 24 * time.Hour

var (
	reEmail          = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
	errCredenciais   = apperrors.Unauthorized("CREDENCIAIS_INVALIDAS", "E-mail ou senha incorretos.")
	errNaoAutenticad = apperrors.Unauthorized("NAO_AUTENTICADO", "Entre na sua conta para continuar.")
	// hash de uma senha qualquer: o login de e-mail inexistente gasta o mesmo tempo que o de senha errada
	hashFicticio, _ = bcrypt.GenerateFromPassword([]byte("antecipe-senha-ficticia"), bcrypt.DefaultCost)
)

type Servico struct {
	repo domain.UsuarioRepository
	log  *slog.Logger
}

func Novo(repo domain.UsuarioRepository, log *slog.Logger) *Servico {
	return &Servico{repo: repo, log: log}
}

// Cadastro são os dados que a empresa informa.
type Cadastro struct {
	Nome, Email, Senha, EmpresaNome, EmpresaCNPJ string
}

type problema struct {
	Campo    string `json:"campo"`
	Problema string `json:"problema"`
}

func normalizarEmail(e string) string { return strings.ToLower(strings.TrimSpace(e)) }

func validarBase(nome, email, senha string) []problema {
	var ps []problema
	if n := utf8.RuneCountInString(strings.TrimSpace(nome)); n < 2 || n > 100 {
		ps = append(ps, problema{"nome", "informe seu nome (2 a 100 caracteres)"})
	}
	if len(email) > 200 || !reEmail.MatchString(email) {
		ps = append(ps, problema{"email", "não é um e-mail válido"})
	}
	if len(senha) < 8 || len(senha) > 72 {
		ps = append(ps, problema{"senha", "a senha precisa ter de 8 a 72 caracteres"})
	}
	return ps
}

func erroCadastro(ps []problema) error {
	return apperrors.Validation("CADASTRO_INVALIDO", "Confira os dados do cadastro.").WithDetails(ps)
}

// Cadastrar cria uma conta de empresa (o cadastro aberto nunca cria gestor) e já abre a sessão.
func (s *Servico) Cadastrar(ctx context.Context, c Cadastro) (domain.Usuario, string, error) {
	email := normalizarEmail(c.Email)
	ps := validarBase(c.Nome, email, c.Senha)
	if n := utf8.RuneCountInString(strings.TrimSpace(c.EmpresaNome)); n < 2 || n > 150 {
		ps = append(ps, problema{"empresa_nome", "informe o nome da empresa (2 a 150 caracteres)"})
	}
	cnpj := document.Normalizar(c.EmpresaCNPJ)
	if !document.CNPJValido(cnpj) {
		ps = append(ps, problema{"empresa_cnpj", "não é um CNPJ válido"})
	}
	if len(ps) > 0 {
		return domain.Usuario{}, "", erroCadastro(ps)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(c.Senha), bcrypt.DefaultCost)
	if err != nil {
		return domain.Usuario{}, "", err
	}
	u, err := s.repo.CriarUsuario(ctx, domain.NovoUsuario{
		Email: email, Nome: strings.TrimSpace(c.Nome), Perfil: domain.PerfilEmpresa,
		EmpresaNome: strings.TrimSpace(c.EmpresaNome), EmpresaCNPJ: document.Formatar(cnpj), SenhaHash: string(hash),
	})
	if err != nil {
		return u, "", err
	}
	token, err := s.abrirSessao(ctx, u.ID)
	if err == nil {
		s.log.InfoContext(ctx, "conta criada", slog.String("perfil", u.Perfil))
	}
	return u, token, err
}

// CriarGestor cria outro gestor (só um gestor logado chama isso; o handler confere).
func (s *Servico) CriarGestor(ctx context.Context, nome, email, senha string) (domain.Usuario, error) {
	email = normalizarEmail(email)
	if ps := validarBase(nome, email, senha); len(ps) > 0 {
		return domain.Usuario{}, erroCadastro(ps)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)
	if err != nil {
		return domain.Usuario{}, err
	}
	u, err := s.repo.CriarUsuario(ctx, domain.NovoUsuario{Email: email, Nome: strings.TrimSpace(nome), Perfil: domain.PerfilGestor, SenhaHash: string(hash)})
	if err == nil {
		s.log.InfoContext(ctx, "gestor criado")
	}
	return u, err
}

// GarantirGestorInicial cria (ou atualiza a senha de) o gestor definido no .env.
func (s *Servico) GarantirGestorInicial(ctx context.Context, email, senha string) error {
	email = normalizarEmail(email)
	if email == "" || senha == "" {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.GarantirGestor(ctx, domain.NovoUsuario{Email: email, Nome: "Gestor do fundo", SenhaHash: string(hash)})
}

// Entrar confere e-mail e senha e abre uma sessão.
func (s *Servico) Entrar(ctx context.Context, email, senha string) (domain.Usuario, string, error) {
	u, hash, err := s.repo.CredenciaisPorEmail(ctx, normalizarEmail(email))
	if err != nil {
		_ = bcrypt.CompareHashAndPassword(hashFicticio, []byte(senha))
		if apperrors.IsCode(err, "CREDENCIAIS_INVALIDAS") {
			return u, "", errCredenciais
		}
		return u, "", err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(senha)) != nil {
		return domain.Usuario{}, "", errCredenciais
	}
	token, err := s.abrirSessao(ctx, u.ID)
	return u, token, err
}

// Sair encerra a sessão do token.
func (s *Servico) Sair(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.repo.ApagarSessao(ctx, hashToken(token))
}

// UsuarioDoToken devolve o dono de um token válido.
func (s *Servico) UsuarioDoToken(ctx context.Context, token string) (domain.Usuario, error) {
	if token == "" || len(token) > 200 {
		return domain.Usuario{}, errNaoAutenticad
	}
	return s.repo.UsuarioDaSessao(ctx, hashToken(token))
}

// ListarLotes devolve os lotes da conta; o gestor vê todos.
func (s *Servico) ListarLotes(ctx context.Context, u domain.Usuario) ([]domain.ItemLote, error) {
	dono := u.ID
	if u.Perfil == domain.PerfilGestor {
		dono = ""
	}
	return s.repo.ListarLotes(ctx, dono, 200)
}

func (s *Servico) abrirSessao(ctx context.Context, usuarioID string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(b)
	return token, s.repo.CriarSessao(ctx, hashToken(token), usuarioID, time.Now().Add(DuracaoSessao))
}

func hashToken(t string) string {
	h := sha256.Sum256([]byte(t))
	return hex.EncodeToString(h[:])
}
