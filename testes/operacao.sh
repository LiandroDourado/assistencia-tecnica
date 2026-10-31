#!/bin/bash
# Testes de operação do executável (comportamento do container).
# Requer Linux e root (para testar a redução de privilégios). Uso: sudo BINARIO=./assistencia testes/operacao.sh
B=${BINARIO:-./assistencia}; ok=0; falha=0
passou(){ echo "  ok    $1"; ok=$((ok+1)); }
falhou(){ echo "  FALHA $1"; falha=$((falha+1)); }
esperar(){ for i in $(seq 50); do curl -sf "http://127.0.0.1:$1/api/saude" >/dev/null && return 0; sleep 0.1; done; return 1; }

D=$(mktemp -d); chown root:root $D
# 1. Começa como root, acerta permissões e roda como 1000
DIR_DADOS=$D PORTA=18101 PUID=1000 PGID=1000 $B > $D.log 2>&1 & P=$!
esperar 18101 && [ "$(stat -c %u $D/db/assistencia.db)" = 1000 ] && [ "$(ps -o uid= -p $P | tr -d ' ')" = 1000 ] \
  && passou "inicia como root, ajusta a pasta e roda como UID 1000" || falhou "redução de privilégios"
# 2. Healthcheck responde 0 com o servidor no ar
PORTA=18101 $B healthcheck && passou "healthcheck = 0 com o servidor no ar" || falhou "healthcheck no ar"
# 3. SIGTERM (docker stop) encerra limpo
kill -TERM $P; wait $P; rc=$?
[ $rc = 0 ] && grep -q "servidor encerrado" $D.log && passou "docker stop (SIGTERM) encerra limpo, código 0" || falhou "SIGTERM: código $rc"
# 4. Healthcheck responde 1 com o servidor fora
PORTA=18101 $B healthcheck 2>/dev/null && falhou "healthcheck fora do ar deveria falhar" || passou "healthcheck = 1 com o servidor fora"
# 5. Porta ocupada: mensagem clara e código 1
python3 -c "import socket,time;s=socket.socket();s.bind(('0.0.0.0',18102));s.listen();time.sleep(4)" & S=$!; sleep 0.5
DIR_DADOS=$D PORTA=18102 $B > $D.log2 2>&1; rc=$?
kill $S 2>/dev/null
[ $rc = 1 ] && grep -q "porta 18102" $D.log2 && passou "porta ocupada: sai com código 1 e explica" || { falhou "porta ocupada (rc=$rc)"; cat $D.log2; }
# 6. PUID inválido
DIR_DADOS=$D PORTA=18103 PUID=abc $B > $D.log3 2>&1; rc=$?
[ $rc = 1 ] && grep -q "PUID/PGID inválidos" $D.log3 && passou "PUID inválido: sai com código 1 e explica" || falhou "PUID inválido (rc=$rc)"
# 7. Pasta de dados sem permissão (usuário comum)
R=$(mktemp -d); chmod 500 $R; chown root:root $R
su -s /bin/sh nobody -c "DIR_DADOS=$R/x PORTA=18104 $B" > $D.log4 2>&1; rc=$?
[ $rc = 1 ] && grep -q "não foi possível criar a pasta de dados" $D.log4 && passou "pasta sem permissão: sai com código 1 e explica" || { falhou "pasta sem permissão (rc=$rc)"; cat $D.log4; }
# 8. Reinício preserva dados; a migração não é reaplicada
DIR_DADOS=$D PORTA=18105 $B > $D.log5 2>&1 & P=$!; esperar 18105
curl -s -c $D.c -H 'Content-Type: application/json' -d '{"nome":"A","email":"a@loja.com","senha":"senha-forte-123"}' localhost:18105/api/primeiro-acesso >/dev/null
kill -TERM $P; wait $P
DIR_DADOS=$D PORTA=18105 $B > $D.log6 2>&1 & P=$!; esperar 18105
cfg=$(curl -s localhost:18105/api/estado); eu=$(curl -s -b $D.c localhost:18105/api/auth/eu)
echo "$cfg" | grep -q '"configurado":true' && echo "$eu" | grep -q a@loja.com && passou "reinício preserva usuários e sessões" || falhou "reinício: $cfg $eu"
# 9. Memória em repouso
rss=$(ps -o rss= -p $P); [ $rss -lt 60000 ] && passou "memória em repouso: $((rss/1024)) MB" || falhou "memória em repouso alta: $((rss/1024)) MB"
# 10. Carga: 40 logins simultâneos (Argon2) + 400 leituras simultâneas, sem erro e com memória contida
python3 - <<PY
import concurrent.futures as cf, json, urllib.request, http.cookiejar
B="http://127.0.0.1:18105"
def req(m,c,corpo=None,cookie=None):
    r=urllib.request.Request(B+c,method=m,data=json.dumps(corpo).encode() if corpo else None,headers={"Content-Type":"application/json"})
    if cookie: r.add_header("Cookie",cookie)
    try:
        with urllib.request.urlopen(r,timeout=30) as x: return x.status, x.headers.get("Set-Cookie","")
    except urllib.error.HTTPError as e: return e.code, ""
st,ck=req("POST","/api/auth/entrar",{"email":"a@loja.com","senha":"senha-forte-123"}); ck=ck.split(";")[0]
with cf.ThreadPoolExecutor(40) as ex:
    logins=list(ex.map(lambda _: req("POST","/api/auth/entrar",{"email":"a@loja.com","senha":"senha-forte-123"})[0], range(40)))
with cf.ThreadPoolExecutor(50) as ex:
    leituras=list(ex.map(lambda _: req("GET","/api/auth/eu",cookie=ck)[0], range(400)))
with cf.ThreadPoolExecutor(20) as ex:
    criados=list(ex.map(lambda i: req("POST","/api/usuarios",{"nome":f"U{i}","email":f"u{i}@loja.com","senha":"senha-forte-123","perfil":"balcao"},cookie=ck)[0], range(20)))
print("logins", set(logins), "leituras", set(leituras), "criacoes", set(criados))
open("/tmp/carga.ok","w").write("1" if set(logins)=={200} and set(leituras)=={200} and set(criados)=={201} else "0")
PY
rss=$(ps -o rss= -p $P)
[ "$(cat /tmp/carga.ok)" = 1 ] && passou "carga: 40 logins + 400 leituras + 20 gravações simultâneas sem erro" || falhou "carga com erros"
[ $rss -lt 120000 ] && passou "memória após a carga: $((rss/1024)) MB" || falhou "memória após a carga: $((rss/1024)) MB"
grep -qi "database is locked\|SQLITE_BUSY" $D.log6 && falhou "banco ocupado no log" || passou "nenhum 'banco ocupado' no log"
kill -TERM $P; wait $P
echo; echo "$ok passaram, $falha falharam"; [ $falha = 0 ]
