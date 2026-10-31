package servidor

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// Erros sempre em português, no formato {"codigo": "...", "erro": "..."}.
func responderErro(w http.ResponseWriter, status int, codigo, mensagem string) {
	responderJSON(w, status, map[string]string{"codigo": codigo, "erro": mensagem})
}

func responderJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func erroValidacao(w http.ResponseWriter, mensagem string) {
	responderErro(w, http.StatusUnprocessableEntity, "VALIDACAO", mensagem)
}

func erroInterno(w http.ResponseWriter, err error) {
	slog.Error("erro interno", "erro", err)
	responderErro(w, http.StatusInternalServerError, "INTERNO",
		"Erro interno do servidor. Tente novamente; se persistir, avise o administrador.")
}

func semPermissao(w http.ResponseWriter) {
	responderErro(w, http.StatusForbidden, "SEM_PERMISSAO", "Seu perfil não tem permissão para esta ação.")
}

const limiteCorpo = 1 << 20 // 1 MB

// lerJSON decodifica o corpo, com mensagens de erro em português.
func lerJSON(w http.ResponseWriter, r *http.Request, destino any) bool {
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		erroValidacao(w, "Envie os dados no formato JSON.")
		return false
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, limiteCorpo))
	if err := dec.Decode(destino); err != nil {
		var maxErr *http.MaxBytesError
		switch {
		case errors.As(err, &maxErr):
			erroValidacao(w, "Dados grandes demais.")
		case errors.Is(err, io.EOF):
			erroValidacao(w, "Nenhum dado foi enviado.")
		default:
			var tipo *json.UnmarshalTypeError
			if errors.As(err, &tipo) {
				erroValidacao(w, "Valor inválido no campo "+tipo.Field+".")
			} else {
				erroValidacao(w, "JSON malformado.")
			}
		}
		return false
	}
	return true
}
