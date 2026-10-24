// Package apperrors define os erros da aplicação: um código estável, uma mensagem em pt-BR
// que pode ir para o usuário, o status HTTP e detalhes opcionais. A causa técnica (Err)
// nunca é exposta na resposta; erros desconhecidos viram ERRO_INTERNO.
package apperrors

import (
	"errors"
	"net/http"
)

// Error é um erro conhecido da aplicação.
type Error struct {
	Status  int    // status HTTP
	Code    string // ex.: LOTE_NAO_ENCONTRADO
	Message string // texto para o usuário
	Details any    // ex.: lista de campos inválidos
	Err     error  // causa técnica (só para logs e errors.Is)
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Code + ": " + e.Message + ": " + e.Err.Error()
	}
	return e.Code + ": " + e.Message
}

func (e *Error) Unwrap() error { return e.Err }

// WithDetails devolve uma cópia com detalhes.
func (e *Error) WithDetails(d any) *Error {
	c := *e
	c.Details = d
	return &c
}

// Wrap devolve uma cópia que guarda a causa técnica.
func (e *Error) Wrap(err error) *Error {
	c := *e
	c.Err = err
	return &c
}

func New(status int, code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

func BadRequest(code, msg string) *Error   { return New(http.StatusBadRequest, code, msg) }
func Unauthorized(code, msg string) *Error { return New(http.StatusUnauthorized, code, msg) }
func NotFound(code, msg string) *Error     { return New(http.StatusNotFound, code, msg) }
func Conflict(code, msg string) *Error     { return New(http.StatusConflict, code, msg) }
func TooLarge(code, msg string) *Error     { return New(http.StatusRequestEntityTooLarge, code, msg) }
func Unsupported(code, msg string) *Error  { return New(http.StatusUnsupportedMediaType, code, msg) }
func Validation(code, msg string) *Error   { return New(http.StatusUnprocessableEntity, code, msg) }
func Unavailable(code, msg string) *Error  { return New(http.StatusServiceUnavailable, code, msg) }

// Internal embrulha um erro inesperado sem expor o texto dele.
func Internal(err error) *Error {
	return &Error{Status: http.StatusInternalServerError, Code: "ERRO_INTERNO", Message: "Ocorreu um erro inesperado. Tente de novo.", Err: err}
}

// From devolve o *Error da cadeia, ou ERRO_INTERNO para erros desconhecidos.
func From(err error) *Error {
	if err == nil {
		return nil
	}
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return Internal(err)
}

// IsCode diz se a cadeia de erros contém um *Error com o código dado.
func IsCode(err error, code string) bool {
	var e *Error
	return errors.As(err, &e) && e.Code == code
}
