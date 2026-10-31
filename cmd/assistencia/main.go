// Sistema de Assistência Técnica — executável único: API + interface.
//
//	assistencia              inicia o servidor
//	assistencia healthcheck  usado pelo Docker para saber se o servidor responde
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
	_ "time/tzdata" // fuso horário funciona mesmo sem tzdata no sistema

	"github.com/liandrodourado/assistencia-tecnica/internal/banco"
	"github.com/liandrodourado/assistencia-tecnica/internal/estatico"
	"github.com/liandrodourado/assistencia-tecnica/internal/servidor"
)

// versao é preenchida na compilação: -ldflags "-X main.versao=1.0.0"
var versao = "desenvolvimento"

func env(nome, padrao string) string {
	if v := os.Getenv(nome); v != "" {
		return v
	}
	return padrao
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	porta := env("PORTA", "8089")

	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck(porta))
	}
	if err := executar(porta); err != nil {
		slog.Error("falha ao iniciar", "erro", err)
		os.Exit(1)
	}
}

func healthcheck(porta string) int {
	c := http.Client{Timeout: 4 * time.Second}
	r, err := c.Get("http://127.0.0.1:" + porta + "/api/saude")
	if err != nil {
		fmt.Fprintln(os.Stderr, "sem resposta:", err)
		return 1
	}
	r.Body.Close()
	if r.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, "status", r.StatusCode)
		return 1
	}
	return 0
}

func executar(porta string) error {
	dados := env("DIR_DADOS", "./dados")
	for _, sub := range []string{"db", "media/os", "media/produtos", "backups/diario", "backups/semanal", "backups/manual", "config"} {
		if err := os.MkdirAll(filepath.Join(dados, sub), 0o750); err != nil {
			return fmt.Errorf("não foi possível criar a pasta de dados %s: %w", dados, err)
		}
	}
	if err := reduzirPrivilegios(dados); err != nil {
		return err
	}

	ctx, parar := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer parar()

	db, err := banco.Abrir(ctx, filepath.Join(dados, "db", "assistencia.db"))
	if err != nil {
		return fmt.Errorf("banco de dados: %w", err)
	}
	defer db.Fechar()

	srv := servidor.Novo(db, estatico.App(), versao, env("COOKIE_SEGURO", "false") == "true")
	go func() {
		t := time.NewTicker(time.Hour)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				srv.LimparSessoes(ctx)
			}
		}
	}()

	httpSrv := &http.Server{
		Addr:              ":" + porta,
		Handler:           srv.Rotas(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	erroServidor := make(chan error, 1)
	go func() {
		slog.Info("servidor iniciado", "versao", versao, "porta", porta, "dados", dados)
		if err := httpSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			erroServidor <- err
		}
	}()

	select {
	case err := <-erroServidor:
		return fmt.Errorf("porta %s: %w", porta, err)
	case <-ctx.Done():
	}
	slog.Info("encerrando: aguardando requisições em andamento")
	fim, cancelar := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelar()
	httpSrv.Shutdown(fim)
	slog.Info("servidor encerrado")
	return nil
}
