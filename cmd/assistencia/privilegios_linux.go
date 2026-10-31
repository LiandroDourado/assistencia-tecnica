package main

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
)

// reduzirPrivilegios: o container começa como root só para acertar o dono da
// pasta de dados (no ZimaOS, /DATA/AppData costuma pertencer ao root) e logo
// passa a rodar como PUID:PGID. Fora do root, não faz nada.
func reduzirPrivilegios(dados string) error {
	if os.Getuid() != 0 {
		return nil
	}
	uid, err1 := strconv.Atoi(env("PUID", "1000"))
	gid, err2 := strconv.Atoi(env("PGID", "1000"))
	if err1 != nil || err2 != nil || uid < 0 || gid < 0 {
		return fmt.Errorf("PUID/PGID inválidos")
	}
	if uid == 0 {
		slog.Warn("PUID=0: o sistema vai rodar como root, o que não é recomendado")
		return nil
	}
	err := filepath.WalkDir(dados, func(caminho string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if st, ok := info.Sys().(*syscall.Stat_t); ok && (int(st.Uid) != uid || int(st.Gid) != gid) {
			return os.Lchown(caminho, uid, gid)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("ajustar permissões de %s: %w", dados, err)
	}
	if err := syscall.Setgroups([]int{}); err != nil {
		return fmt.Errorf("setgroups: %w", err)
	}
	if err := syscall.Setgid(gid); err != nil {
		return fmt.Errorf("setgid: %w", err)
	}
	if err := syscall.Setuid(uid); err != nil {
		return fmt.Errorf("setuid: %w", err)
	}
	slog.Info("executando sem privilégios de root", "uid", uid, "gid", gid)
	return nil
}
