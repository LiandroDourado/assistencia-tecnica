// Package seguranca cuida de senhas (Argon2id) e tokens de sessão.
package seguranca

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/crypto/argon2"
)

// Parâmetros recomendados pela OWASP para Argon2id: 19 MiB, 2 passadas, 1 thread.
const (
	memoriaKiB = 19 * 1024
	passadas   = 2
	threads    = 1
	tamChave   = 32
	tamSal     = 16
)

// No máximo 2 cálculos de senha ao mesmo tempo: cada um usa 19 MiB, e uma
// rajada de logins não pode consumir a memória do servidor.
var limite = make(chan struct{}, 2)

func comLimite[T any](f func() T) T {
	limite <- struct{}{}
	defer func() { <-limite }()
	return f()
}

var b64 = base64.RawStdEncoding

// HashSenha devolve a senha no formato PHC: $argon2id$v=19$m=...,t=...,p=...$sal$hash
func HashSenha(senha string) (string, error) {
	sal := make([]byte, tamSal)
	if _, err := rand.Read(sal); err != nil {
		return "", err
	}
	h := comLimite(func() []byte {
		return argon2.IDKey([]byte(senha), sal, passadas, memoriaKiB, threads, tamChave)
	})
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, memoriaKiB, passadas, threads, b64.EncodeToString(sal), b64.EncodeToString(h)), nil
}

var errFormato = errors.New("hash em formato desconhecido")

// SenhaConfere compara em tempo constante. Hash inválido nunca confere.
func SenhaConfere(senha, hash string) bool {
	ok, err := confere(senha, hash)
	return err == nil && ok
}

func confere(senha, hash string) (bool, error) {
	p := strings.Split(hash, "$")
	if len(p) != 6 || p[1] != "argon2id" {
		return false, errFormato
	}
	var m, t uint32
	var th uint8
	if _, err := fmt.Sscanf(p[3], "m=%d,t=%d,p=%d", &m, &t, &th); err != nil {
		return false, errFormato
	}
	sal, err1 := b64.DecodeString(p[4])
	esperado, err2 := b64.DecodeString(p[5])
	if err1 != nil || err2 != nil || len(esperado) == 0 {
		return false, errFormato
	}
	calc := comLimite(func() []byte {
		return argon2.IDKey([]byte(senha), sal, t, m, th, uint32(len(esperado)))
	})
	return subtle.ConstantTimeCompare(calc, esperado) == 1, nil
}

var (
	ficticioUma sync.Once
	ficticio    string
)

// ConsumirTempo faz o mesmo trabalho de uma verificação de senha. Usado quando
// o e-mail não existe, para a resposta não revelar quem tem cadastro.
func ConsumirTempo(senha string) {
	ficticioUma.Do(func() { ficticio, _ = HashSenha("senha-ficticia-para-tempo-constante") })
	SenhaConfere(senha, ficticio)
}

// NovoToken gera 32 bytes aleatórios em hexadecimal (vai no cookie).
func NovoToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// HashToken é o que fica no banco: quem ler o banco não consegue usar a sessão.
func HashToken(token string) string {
	s := sha256.Sum256([]byte(token))
	return hex.EncodeToString(s[:])
}

// ProblemaSenhaNova aplica a regra mínima de senha. Devolve a mensagem para
// o usuário, ou "" se a senha for aceitável.
func ProblemaSenhaNova(senha string) string {
	if len([]rune(senha)) < 8 {
		return "A senha deve ter pelo menos 8 caracteres."
	}
	if len(senha) > 200 {
		return "A senha deve ter no máximo 200 caracteres."
	}
	return ""
}
