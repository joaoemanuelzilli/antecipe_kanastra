package rest

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"antecipe/internal/domain"
	"antecipe/internal/platform/apperrors"
	"antecipe/internal/service/auth"
)

const nomeCookie = "antecipe_sessao"

type chaveUsuario struct{}

// usuarioDe devolve a conta da requisição, ou nil se ela veio sem login.
func usuarioDe(r *http.Request) *domain.Usuario {
	u, _ := r.Context().Value(chaveUsuario{}).(*domain.Usuario)
	return u
}

func tokenDe(r *http.Request) string {
	if a := r.Header.Get("Authorization"); strings.HasPrefix(a, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(a, "Bearer "))
	}
	if c, err := r.Cookie(nomeCookie); err == nil {
		return c.Value
	}
	return ""
}

// autenticar coloca a conta no contexto quando há sessão válida; sem sessão, segue anônimo.
func (h *Handlers) autenticar(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if h.d.Auth != nil {
			if t := tokenDe(r); t != "" {
				if u, err := h.d.Auth.UsuarioDoToken(r.Context(), t); err == nil {
					r = r.WithContext(context.WithValue(r.Context(), chaveUsuario{}, &u))
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

func definirCookie(w http.ResponseWriter, r *http.Request, token string, idade time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name: nomeCookie, Value: token, Path: "/", HttpOnly: true, SameSite: http.SameSiteLaxMode,
		Secure: r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https", MaxAge: int(idade.Seconds()),
	})
}

type usuarioDTO struct {
	ID          string    `json:"id"`
	Nome        string    `json:"nome"`
	Email       string    `json:"email"`
	Perfil      string    `json:"perfil"`
	EmpresaNome string    `json:"empresa_nome,omitempty"`
	EmpresaCNPJ string    `json:"empresa_cnpj,omitempty"`
	CriadoEm    time.Time `json:"criado_em"`
}

func paraUsuario(u domain.Usuario) usuarioDTO {
	return usuarioDTO{ID: u.ID, Nome: u.Nome, Email: u.Email, Perfil: u.Perfil, EmpresaNome: u.EmpresaNome, EmpresaCNPJ: u.EmpresaCNPJ, CriadoEm: u.CriadoEm}
}

func lerCorpoJSON(w http.ResponseWriter, r *http.Request, v any) error {
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
	if err != nil || json.Unmarshal(b, v) != nil {
		return apperrors.BadRequest("JSON_INVALIDO", "Corpo inválido: envie um JSON.")
	}
	return nil
}

func (h *Handlers) cadastrar(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Nome        string `json:"nome"`
		Email       string `json:"email"`
		Senha       string `json:"senha"`
		EmpresaNome string `json:"empresa_nome"`
		EmpresaCNPJ string `json:"empresa_cnpj"`
	}
	if err := lerCorpoJSON(w, r, &in); err != nil {
		EscreverErro(w, r, h.d.Log, err)
		return
	}
	u, token, err := h.d.Auth.Cadastrar(r.Context(), auth.Cadastro{Nome: in.Nome, Email: in.Email, Senha: in.Senha, EmpresaNome: in.EmpresaNome, EmpresaCNPJ: in.EmpresaCNPJ})
	if err != nil {
		EscreverErro(w, r, h.d.Log, err)
		return
	}
	definirCookie(w, r, token, auth.DuracaoSessao)
	EscreverJSON(w, http.StatusCreated, map[string]any{"usuario": paraUsuario(u)})
}

func (h *Handlers) entrar(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email string `json:"email"`
		Senha string `json:"senha"`
	}
	if err := lerCorpoJSON(w, r, &in); err != nil {
		EscreverErro(w, r, h.d.Log, err)
		return
	}
	u, token, err := h.d.Auth.Entrar(r.Context(), in.Email, in.Senha)
	if err != nil {
		EscreverErro(w, r, h.d.Log, err)
		return
	}
	definirCookie(w, r, token, auth.DuracaoSessao)
	// o token também volta no corpo para quem usa a API direto (Authorization: Bearer); o navegador usa o cookie
	EscreverJSON(w, http.StatusOK, map[string]any{"usuario": paraUsuario(u), "token": token, "expira_em": time.Now().Add(auth.DuracaoSessao)})
}

func (h *Handlers) sair(w http.ResponseWriter, r *http.Request) {
	_ = h.d.Auth.Sair(r.Context(), tokenDe(r))
	definirCookie(w, r, "", -time.Second)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) eu(w http.ResponseWriter, r *http.Request) {
	u := usuarioDe(r)
	if u == nil {
		EscreverErro(w, r, h.d.Log, apperrors.Unauthorized("NAO_AUTENTICADO", "Entre na sua conta para continuar."))
		return
	}
	EscreverJSON(w, http.StatusOK, map[string]any{"usuario": paraUsuario(*u)})
}

func (h *Handlers) criarGestor(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Nome  string `json:"nome"`
		Email string `json:"email"`
		Senha string `json:"senha"`
	}
	if err := lerCorpoJSON(w, r, &in); err != nil {
		EscreverErro(w, r, h.d.Log, err)
		return
	}
	u, err := h.d.Auth.CriarGestor(r.Context(), in.Nome, in.Email, in.Senha)
	if err != nil {
		EscreverErro(w, r, h.d.Log, err)
		return
	}
	EscreverJSON(w, http.StatusCreated, map[string]any{"usuario": paraUsuario(u)})
}

// listarLotes: "meus lotes" (a empresa vê os dela; o gestor vê todos).
func (h *Handlers) listarLotes(w http.ResponseWriter, r *http.Request) {
	u := usuarioDe(r)
	if u == nil {
		EscreverErro(w, r, h.d.Log, apperrors.Unauthorized("NAO_AUTENTICADO", "Entre na sua conta para ver seus lotes."))
		return
	}
	itens, err := h.d.Auth.ListarLotes(r.Context(), *u)
	if err != nil {
		EscreverErro(w, r, h.d.Log, err)
		return
	}
	out := make([]map[string]any, 0, len(itens))
	for _, it := range itens {
		out = append(out, map[string]any{"id": it.ID, "status": it.Status, "data_base": it.DataBase.Format("2006-01-02"),
			"criado_em": it.CriadoEm, "concluido_em": it.ConcluidoEm, "totais": it.Totais, "cedido": it.TemCessao})
	}
	EscreverJSON(w, http.StatusOK, map[string]any{"lotes": out})
}

// podeVer: lote anônimo é visível por quem tem o id; lote de conta, só pela dona e pelos gestores.
func podeVer(l domain.Lote, u *domain.Usuario) bool {
	if l.UsuarioID == "" {
		return true
	}
	return u != nil && (u.Perfil == domain.PerfilGestor || u.ID == l.UsuarioID)
}

var errLoteNaoEncontrado = apperrors.NotFound("LOTE_NAO_ENCONTRADO", "Lote não encontrado.")

// loteVisivel busca o lote e aplica a regra de visibilidade (404 para quem não pode ver, sem revelar que existe).
func (h *Handlers) loteVisivel(r *http.Request, id string) (domain.Lote, int, error) {
	l, pos, err := h.d.Lotes.Obter(r.Context(), id)
	if err != nil {
		return l, 0, err
	}
	if !podeVer(l, usuarioDe(r)) {
		return domain.Lote{}, 0, errLoteNaoEncontrado
	}
	return l, pos, nil
}

// exigirLoginSeConfigurado barra envios anônimos quando exigir_login está ligado no YAML.
func (h *Handlers) exigirLoginSeConfigurado(r *http.Request) error {
	if h.d.Config != nil && h.d.Config.Obter().Processamento.ExigirLogin && usuarioDe(r) == nil {
		return apperrors.Unauthorized("NAO_AUTENTICADO", "Entre na sua conta para enviar lotes.")
	}
	return nil
}
