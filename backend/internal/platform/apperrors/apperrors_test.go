package apperrors

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestConstrutores(t *testing.T) {
	casos := []struct {
		err    *Error
		status int
	}{
		{BadRequest("ARQUIVO_AUSENTE", "Envie um arquivo."), http.StatusBadRequest},
		{Unauthorized("SENHA_INVALIDA", "Senha inválida."), http.StatusUnauthorized},
		{NotFound("LOTE_NAO_ENCONTRADO", "Lote não encontrado."), http.StatusNotFound},
		{Conflict("SEM_APROVADAS", "Nenhuma nota aprovada."), http.StatusConflict},
		{TooLarge("ARQUIVO_GRANDE_DEMAIS", "Arquivo grande demais."), http.StatusRequestEntityTooLarge},
		{Unsupported("ARQUIVO_NAO_CSV", "Não é CSV."), http.StatusUnsupportedMediaType},
		{Validation("CABECALHO_INVALIDO", "Faltou coluna."), http.StatusUnprocessableEntity},
		{Unavailable("BANCO_FORA", "Banco fora do ar."), http.StatusServiceUnavailable},
	}
	for _, c := range casos {
		if c.err.Status != c.status || c.err.Code == "" || c.err.Message == "" {
			t.Errorf("%+v: status %d", c.err, c.status)
		}
	}
}

func TestDetalhesECausa(t *testing.T) {
	causa := errors.New("falha de rede")
	e := Validation("REGRAS_INVALIDAS", "Regras inválidas.").WithDetails([]string{"x"}).Wrap(causa)
	if !errors.Is(e, causa) {
		t.Fatal("Unwrap não devolve a causa")
	}
	if e.Details == nil {
		t.Fatal("detalhes perdidos")
	}
	envolvido := fmt.Errorf("camada de cima: %w", e)
	if got := From(envolvido); got.Code != "REGRAS_INVALIDAS" || got.Status != http.StatusUnprocessableEntity {
		t.Fatalf("From = %+v", got)
	}
}

func TestFromErroGenericoViraInterno(t *testing.T) {
	got := From(errors.New("pq: senha do banco xyz incorreta"))
	if got.Status != http.StatusInternalServerError || got.Code != "ERRO_INTERNO" {
		t.Fatalf("From = %+v", got)
	}
	if got.Message != "Ocorreu um erro inesperado. Tente de novo." {
		t.Fatalf("mensagem vazou detalhe interno: %q", got.Message)
	}
	if From(nil) != nil {
		t.Fatal("From(nil) deveria ser nil")
	}
}

func TestIsCode(t *testing.T) {
	err := fmt.Errorf("x: %w", NotFound("LOTE_NAO_ENCONTRADO", "Lote não encontrado."))
	if !IsCode(err, "LOTE_NAO_ENCONTRADO") || IsCode(err, "OUTRO") || IsCode(errors.New("x"), "LOTE_NAO_ENCONTRADO") {
		t.Fatal("IsCode errado")
	}
}
