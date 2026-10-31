# Sistema de Assistência Técnica

Gestão de assistência técnica para a rede local da loja: ordens de serviço, estoque, vendas e caixa.
Um único container no ZimaOS, instalado por um arquivo YAML.

**Versão 1.0.0 — entrega 1:** instalação, primeiro acesso, login, usuários e perfis.

## Instalação

São duas partes: publicar a imagem no GitHub (uma vez por versão) e instalar no ZimaOS.

### Parte 1 — Publicar a imagem no GitHub

1. Entre em [github.com](https://github.com) com a conta **LiandroDourado**.
2. Clique em **New repository** (botão verde, ou o **+** no canto superior direito).
   - Nome: `assistencia-tecnica`
   - Marque **Public**. Assim o ícone aparece no ZimaOS. O código não contém senhas nem dados da loja.
   - Não marque nenhuma opção de README, .gitignore ou licença.
   - Clique em **Create repository**.
3. Na página do repositório vazio, clique no link **uploading an existing file**.
4. Descompacte o arquivo do projeto no computador, abra a pasta `assistencia-tecnica`, selecione **tudo o que está dentro dela** e arraste para a página do GitHub. Confira se a pasta `.github` foi junto: ela contém a automação que gera a imagem.
5. Clique em **Commit changes** e aguarde o envio terminar.
6. No menu da direita do repositório, clique em **Releases** → **Create a new release**.
   - Em **Choose a tag**, digite `v1.0.0` e clique em **Create new tag**.
   - Título: `1.0.0`. Clique em **Publish release**.
7. Abra a aba **Actions**. A tarefa **Publicar imagem** começa sozinha e leva de 3 a 6 minutos. Espere aparecer o sinal verde.
8. Torne a imagem pública (só na primeira vez):
   - Clique na sua foto (canto superior direito) → **Your profile** → aba **Packages** → **assistencia-tecnica**.
   - **Package settings** (no fim da coluna da direita) → **Change visibility** → **Public** → digite o nome para confirmar.

Se a pasta `.github` não tiver subido no passo 4 (a aba Actions fica vazia), crie o arquivo pelo próprio site:
**Add file** → **Create new file**, nome `.github/workflows/publicar.yml`, cole o conteúdo do arquivo de mesmo nome
do projeto e clique em **Commit changes**. Depois repita o passo 6 com a tag `v1.0.0`.

### Parte 2 — Instalar no ZimaOS

1. Abra o painel do ZimaOS no navegador.
2. Vá em **App Store** → **Instalar app personalizado** (ou o ícone **+** → **Custom Install**).
3. Clique em **Importar** e cole todo o conteúdo do arquivo `docker-compose.yml`.
4. Confira e clique em **Instalar**. O ZimaOS baixa a imagem (cerca de 25 MB).
5. Abra `http://192.168.3.74:8089` no navegador de qualquer aparelho da rede da loja.
6. Na primeira vez aparece a tela **Bem-vindo**: cadastre o nome da loja e o administrador.

### Atualizar para uma nova versão

1. No GitHub, crie uma nova release com a próxima tag (ex.: `v1.1.0`) e espere a aba Actions ficar verde.
2. No ZimaOS, edite o app, troque `1.0.0` pela nova versão na linha `image:` e salve.

Os dados ficam em `/DATA/AppData/assistencia-tecnica/` e não se perdem ao atualizar, reinstalar ou desinstalar.

## Se algo não funcionar

| Sintoma | O que fazer |
| --- | --- |
| ZimaOS diz que não conseguiu baixar a imagem (`denied` ou `unauthorized`) | A imagem ainda está privada: faça o passo 8 da Parte 1 |
| ZimaOS diz `manifest unknown` | A versão no YAML não existe: confira se a aba Actions terminou verde e se a tag é a mesma |
| ZimaOS diz que a porta já está em uso | Outro app usa a 8089: troque o primeiro número em `ports` (ex.: `"8090:8089"`) e acesse pela nova porta |
| A página não abre de outro aparelho | Use `http://` (não `https://`) e o endereço completo com `:8089`; confira se o aparelho está na mesma rede da loja |
| Esqueci a senha do administrador | Outro administrador pode redefinir em **Usuários**. Se não houver outro, apague a pasta `/DATA/AppData/assistencia-tecnica/db` (isso zera o sistema) |

Para ver o que o sistema registrou, abra o app no ZimaOS e consulte os **logs** do container.

## Desenvolvimento

Requer Go 1.26 e Node 22.

```bash
cd frontend && npm install && npm run build && cd ..   # compila a interface para internal/estatico/app
go test ./...                                          # testes
go run ./cmd/assistencia                               # sobe em http://localhost:8089 com dados em ./dados
```

### Testes

| Camada | Comando | O que cobre |
| --- | --- | --- |
| Go | `go test ./...` | Banco, senhas, sessões, permissões e API (roda também dentro do Dockerfile) |
| Interface | `cd frontend && npx svelte-check` | Erros de código e acessibilidade |
| Navegador | `BINARIO=./assistencia python3 testes/e2e_interface.py` | 20 fluxos reais no Chromium, no celular e no computador (requer Playwright) |
| Operação | `sudo BINARIO=./assistencia testes/operacao.sh` | Permissões, healthcheck, desligamento, erros de inicialização, memória e carga |

A automação do GitHub também verifica vulnerabilidades conhecidas nas dependências Go (`govulncheck`) antes de publicar.

Para trabalhar na interface com recarga automática, rode o servidor Go e, em outro terminal, `cd frontend && npm run dev`.

| Variável | Padrão | Uso |
| --- | --- | --- |
| `PORTA` | `8089` | Porta do servidor |
| `DIR_DADOS` | `./dados` (`/dados` na imagem) | Banco, fotos, backups e configurações |
| `PUID` / `PGID` | `1000` | Usuário com que o sistema roda dentro do container |
| `TZ` | `America/Sao_Paulo` | Fuso horário |
| `COOKIE_SEGURO` | `false` | `true` somente se houver HTTPS |

### Estrutura

| Pasta | Conteúdo |
| --- | --- |
| `cmd/assistencia/` | Executável: inicia o servidor, ajusta permissões e responde ao healthcheck |
| `internal/banco/` | SQLite e migrações (`migracoes/*.sql`; nunca altere uma já aplicada, crie a próxima) |
| `internal/seguranca/` | Senhas (Argon2id) e tokens de sessão |
| `internal/servidor/` | Rotas da API e entrega da interface |
| `internal/estatico/` | Interface embutida no executável |
| `frontend/` | Interface em SvelteKit |
| `testes/` | Testes de navegador e de operação |
