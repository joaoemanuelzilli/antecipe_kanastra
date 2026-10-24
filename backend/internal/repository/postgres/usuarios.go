package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"antecipe/internal/domain"
	"antecipe/internal/platform/apperrors"
)

var _ domain.UsuarioRepository = (*Repo)(nil)

var (
	errEmailCadastrado = apperrors.Conflict("EMAIL_JA_CADASTRADO", "Já existe uma conta com esse e-mail.")
	errSemSessao       = apperrors.Unauthorized("NAO_AUTENTICADO", "Entre na sua conta para continuar.")
)

func nuloSeVazio(s string) any {
	if s == "" {
		return nil
	}
	return s
}

const colunasUsuario = `id, email, nome, perfil, COALESCE(empresa_nome, ''), COALESCE(empresa_cnpj, ''), criado_em`

func lerUsuario(row pgx.Row, extra ...any) (domain.Usuario, error) {
	var u domain.Usuario
	dest := append([]any{&u.ID, &u.Email, &u.Nome, &u.Perfil, &u.EmpresaNome, &u.EmpresaCNPJ, &u.CriadoEm}, extra...)
	err := row.Scan(dest...)
	return u, err
}

// CriarUsuario grava a conta; e-mail repetido vira EMAIL_JA_CADASTRADO.
func (r *Repo) CriarUsuario(ctx context.Context, n domain.NovoUsuario) (domain.Usuario, error) {
	u, err := lerUsuario(r.Pool.QueryRow(ctx, `
		INSERT INTO usuarios (email, nome, perfil, empresa_nome, empresa_cnpj, senha_hash)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING `+colunasUsuario,
		n.Email, n.Nome, n.Perfil, nuloSeVazio(n.EmpresaNome), nuloSeVazio(n.EmpresaCNPJ), n.SenhaHash))
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return u, errEmailCadastrado
	}
	return u, err
}

// GarantirGestor cria o gestor do .env ou troca a senha dele, encerrando as sessões abertas.
// Nunca promove uma conta de empresa: se o e-mail for de uma empresa, devolve erro e não mexe em nada.
func (r *Repo) GarantirGestor(ctx context.Context, n domain.NovoUsuario) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background()) //nolint:errcheck
	var id string
	err = tx.QueryRow(ctx, `
		INSERT INTO usuarios (email, nome, perfil, senha_hash) VALUES ($1, $2, 'gestor', $3)
		ON CONFLICT (email) DO UPDATE SET senha_hash = EXCLUDED.senha_hash WHERE usuarios.perfil = 'gestor'
		RETURNING id`, n.Email, n.Nome, n.SenhaHash).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return errors.New("o GESTOR_EMAIL pertence a uma conta de empresa; escolha outro e-mail para o gestor")
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM sessoes WHERE usuario_id = $1`, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// DesativarGestor tranca a conta do gestor do .env (senha impossível) e encerra as sessões dela.
func (r *Repo) DesativarGestor(ctx context.Context, email string) error {
	_, err := r.Pool.Exec(ctx, `
		WITH g AS (UPDATE usuarios SET senha_hash = '!' WHERE email = $1 AND perfil = 'gestor' RETURNING id)
		DELETE FROM sessoes WHERE usuario_id IN (SELECT id FROM g)`, email)
	return err
}

// CredenciaisPorEmail devolve o usuário e o hash da senha (só para o login).
func (r *Repo) CredenciaisPorEmail(ctx context.Context, email string) (domain.Usuario, string, error) {
	var hash string
	u, err := lerUsuario(r.Pool.QueryRow(ctx, `SELECT `+colunasUsuario+`, senha_hash FROM usuarios WHERE email = $1`, email), &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return u, "", apperrors.Unauthorized("CREDENCIAIS_INVALIDAS", "E-mail ou senha incorretos.")
	}
	return u, hash, err
}

// CriarSessao guarda o hash do token (o token em si nunca vai para o banco).
func (r *Repo) CriarSessao(ctx context.Context, tokenHash, usuarioID string, expira time.Time) error {
	_, err := r.Pool.Exec(ctx, `DELETE FROM sessoes WHERE expira_em < now()`)
	if err != nil {
		return err
	}
	_, err = r.Pool.Exec(ctx, `INSERT INTO sessoes (token_hash, usuario_id, expira_em) VALUES ($1, $2, $3)`, tokenHash, usuarioID, expira)
	return err
}

// UsuarioDaSessao devolve o dono de uma sessão válida.
func (r *Repo) UsuarioDaSessao(ctx context.Context, tokenHash string) (domain.Usuario, error) {
	u, err := lerUsuario(r.Pool.QueryRow(ctx, `
		SELECT u.id, u.email, u.nome, u.perfil, COALESCE(u.empresa_nome, ''), COALESCE(u.empresa_cnpj, ''), u.criado_em
		FROM sessoes s JOIN usuarios u ON u.id = s.usuario_id
		WHERE s.token_hash = $1 AND s.expira_em > now()`, tokenHash))
	if errors.Is(err, pgx.ErrNoRows) {
		return u, errSemSessao
	}
	return u, err
}

// ApagarSessao encerra a sessão (logout).
func (r *Repo) ApagarSessao(ctx context.Context, tokenHash string) error {
	_, err := r.Pool.Exec(ctx, `DELETE FROM sessoes WHERE token_hash = $1`, tokenHash)
	return err
}

// ListarLotes devolve os lotes de uma conta (ou de todas, com usuarioID vazio), do mais novo para o mais antigo.
func (r *Repo) ListarLotes(ctx context.Context, usuarioID string, limite int) ([]domain.ItemLote, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT l.id, l.status, l.data_base, l.criado_em, l.concluido_em, l.totais, COALESCE(l.usuario_id::text, ''), c.id IS NOT NULL
		FROM lotes l LEFT JOIN cessoes c ON c.lote_id = l.id
		WHERE ($1::uuid IS NULL OR l.usuario_id = $1::uuid)
		ORDER BY l.criado_em DESC LIMIT $2`, nuloSeVazio(usuarioID), limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.ItemLote
	for rows.Next() {
		var it domain.ItemLote
		var totais []byte
		if err := rows.Scan(&it.ID, &it.Status, &it.DataBase, &it.CriadoEm, &it.ConcluidoEm, &totais, &it.UsuarioID, &it.TemCessao); err != nil {
			return nil, err
		}
		if totais != nil {
			it.Totais = &domain.Totais{}
			_ = json.Unmarshal(totais, it.Totais)
		}
		out = append(out, it)
	}
	return out, rows.Err()
}
