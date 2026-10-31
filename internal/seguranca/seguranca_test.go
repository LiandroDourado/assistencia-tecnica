package seguranca

import (
	"strings"
	"testing"
)

func TestSenha(t *testing.T) {
	h, err := HashSenha("senha-forte-123")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(h, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("formato inesperado: %s", h)
	}
	if !SenhaConfere("senha-forte-123", h) {
		t.Fatal("senha correta não conferiu")
	}
	if SenhaConfere("senha-forte-124", h) {
		t.Fatal("senha errada conferiu")
	}
	h2, _ := HashSenha("senha-forte-123")
	if h == h2 {
		t.Fatal("sal não está variando")
	}
	if SenhaConfere("x", "lixo") || SenhaConfere("x", "$argon2id$v=19$m=1,t=1,p=1$$") {
		t.Fatal("hash inválido conferiu")
	}
}

func TestToken(t *testing.T) {
	a, _ := NovoToken()
	b, _ := NovoToken()
	if len(a) != 64 || a == b {
		t.Fatal("token inválido ou repetido")
	}
	if HashToken(a) == a || len(HashToken(a)) != 64 {
		t.Fatal("hash do token inválido")
	}
}

func TestSenhaNova(t *testing.T) {
	if ProblemaSenhaNova("curta") == "" {
		t.Fatal("aceitou senha curta")
	}
	if ProblemaSenhaNova("çãoçãoçã") != "" { // 8 caracteres com acento
		t.Fatal("contou bytes em vez de caracteres")
	}
	if ProblemaSenhaNova(strings.Repeat("a", 201)) == "" {
		t.Fatal("aceitou senha longa demais")
	}
}
