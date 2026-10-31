package servidor

import (
	"context"
	"database/sql"
	"encoding/json"
)

// auditar grava uma linha imutável no log. antes/depois nunca levam senhas
// nem dados pessoais de clientes.
func auditar(ctx context.Context, tx *sql.Tx, usuarioID *int64, entidade string, entidadeID int64, acao string, antes, depois any) error {
	j := func(v any) any {
		if v == nil {
			return nil
		}
		b, _ := json.Marshal(v)
		return string(b)
	}
	_, err := tx.ExecContext(ctx,
		`INSERT INTO auditoria (usuario_id, entidade, entidade_id, acao, antes, depois) VALUES (?, ?, ?, ?, ?, ?)`,
		usuarioID, entidade, entidadeID, acao, j(antes), j(depois))
	return err
}

func ptr(v int64) *int64 { return &v }
