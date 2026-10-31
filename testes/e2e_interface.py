"""Testes de ponta a ponta da interface, num navegador real (Chromium).

Sobem o executável compilado com um banco vazio e percorrem os fluxos como
uma pessoa usaria. Requer Python 3 com Playwright (pip install playwright;
playwright install chromium).

    BINARIO=/caminho/assistencia python3 testes/e2e_interface.py
"""

import asyncio
import os
import shutil
import signal
import subprocess
import sys
import tempfile
import time
import urllib.request

from playwright.async_api import async_playwright, expect

BINARIO = os.environ.get("BINARIO", "./assistencia")
PORTA = os.environ.get("PORTA_TESTE", "18089")
BASE = f"http://127.0.0.1:{PORTA}"
SENHA = "senha-forte-123"

resultados = []


def subir(dados):
    env = dict(os.environ, DIR_DADOS=dados, PORTA=PORTA)
    proc = subprocess.Popen([BINARIO], env=env, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    for _ in range(50):
        try:
            urllib.request.urlopen(BASE + "/api/saude", timeout=1)
            return proc
        except Exception:
            time.sleep(0.1)
    proc.kill()
    raise SystemExit("o servidor não subiu:\n" + proc.stdout.read().decode())


def parar(proc):
    proc.send_signal(signal.SIGTERM)
    proc.wait(timeout=20)


class Contexto:
    """Uma 'pessoa': navegador isolado, com seus cookies e erros de console."""

    def __init__(self, navegador, **opcoes):
        self.navegador, self.opcoes, self.erros = navegador, opcoes, []

    async def abrir(self):
        self.ctx = await self.navegador.new_context(**self.opcoes)
        self.p = await self.ctx.new_page()
        self.p.on("console", lambda m: m.type == "error" and self.erros.append(m.text))
        self.p.on("pageerror", lambda e: self.erros.append(str(e)))
        return self

    async def entrar(self, email, senha=SENHA):
        await self.p.goto(BASE + "/entrar")
        await self.p.fill("#email", email)
        await self.p.fill("#senha", senha)
        await self.p.click("button[type=submit]")


async def teste(nome, corpo):
    try:
        await corpo()
        resultados.append((True, nome, ""))
        print(f"  ok    {nome}")
    except Exception as e:  # noqa: BLE001
        resultados.append((False, nome, str(e).splitlines()[0][:300]))
        print(f"  FALHA {nome}\n        {str(e).splitlines()[0][:300]}")


async def principal():
    dados = tempfile.mkdtemp(prefix="e2e-")
    proc = subir(dados)
    try:
        async with async_playwright() as pw:
            nav = await pw.chromium.launch()
            # Navegador do sistema preferindo tema CLARO: o sistema deve abrir escuro mesmo assim.
            admin = await Contexto(nav, viewport={"width": 1280, "height": 800}, color_scheme="light").abrir()
            p = admin.p
            expect.set_options(timeout=5000)

            async def primeiro_acesso():
                await p.goto(BASE + "/")
                await expect(p).to_have_url(BASE + "/primeiro-acesso")
                await p.fill("#nome", "Liandro Dourado")
                await p.fill("#email", "liandro@loja.com")
                await p.fill("#senha", SENHA)
                await p.fill("#confirmacao", "outra-coisa")
                await p.click("button[type=submit]")
                await expect(p.locator(".aviso.erro")).to_have_text("As senhas não são iguais.")
                await p.fill("#confirmacao", SENHA)
                await p.fill("#email", "sem-arroba")
                await p.click("button[type=submit]")
                await expect(p.locator(".aviso.erro")).to_have_text("E-mail inválido.")
                await p.fill("#email", "liandro@loja.com")
                await p.fill("#loja", "Conserta Já")
                await p.click("button[type=submit]")
                await expect(p).to_have_url(BASE + "/")
                await expect(p.locator("h1")).to_have_text("Olá, Liandro")
                await expect(p.locator("header .loja")).to_have_text("Conserta Já")
            await teste("primeiro acesso valida e cria o administrador", primeiro_acesso)

            async def tema_escuro_padrao():
                fundo = await p.evaluate("getComputedStyle(document.body).backgroundColor")
                assert fundo == "rgb(26, 31, 38)", f"fundo {fundo}, esperado o escuro"
            await teste("tema escuro é o padrão mesmo com o sistema em claro", tema_escuro_padrao)

            async def tema_persiste():
                await p.click("button.tema")
                await p.reload()
                await expect(p.locator("h1")).to_be_visible()
                fundo = await p.evaluate("getComputedStyle(document.body).backgroundColor")
                assert fundo == "rgb(242, 244, 246)", f"fundo {fundo} após recarregar"
                await p.click("button.tema")
            await teste("escolha do tema claro fica salva", tema_persiste)

            async def primeiro_acesso_fechado():
                await p.goto(BASE + "/primeiro-acesso")
                await expect(p).to_have_url(BASE + "/")
            await teste("primeiro acesso não abre depois de configurado", primeiro_acesso_fechado)

            async def cadastrar_usuarios():
                await p.click("nav >> text=Usuários")
                await expect(p).to_have_url(BASE + "/usuarios")
                for nome, email, perfil in [("Bia Balcão", "bia@loja.com", "balcao"), ("Téo Técnico", "teo@loja.com", "tecnico"),
                                            ("Ana Bloqueio", "ana@loja.com", "balcao")]:
                    await p.click("text=Cadastrar usuário")
                    await p.fill("#u-nome", nome)
                    await p.fill("#u-email", email)
                    await p.select_option("#u-perfil", perfil)
                    await p.fill("#u-senha", SENHA)
                    await p.click("dialog button[type=submit]")
                    await expect(p.locator(".aviso.ok")).to_contain_text(f"{nome} cadastrado")
                await expect(p.locator(".lista li")).to_have_count(4)
            await teste("administrador cadastra usuários", cadastrar_usuarios)

            async def email_duplicado_e_cancelar():
                await p.click("text=Cadastrar usuário")
                await p.fill("#u-nome", "Repetido")
                await p.fill("#u-email", "BIA@loja.com")
                await p.fill("#u-senha", SENHA)
                await p.click("dialog button[type=submit]")
                await expect(p.locator("dialog .aviso.erro")).to_have_text("Este e-mail já está em uso.")
                await p.keyboard.press("Escape")
                await expect(p.locator("dialog")).not_to_be_visible()
                await expect(p.locator(".lista li")).to_have_count(4)
            await teste("e-mail repetido avisa e Esc fecha sem salvar", email_duplicado_e_cancelar)

            async def unico_admin():
                await p.click("li:has-text('Liandro') >> text=Editar")
                await p.select_option("#u-perfil", "balcao")
                await p.click("dialog button[type=submit]")
                await expect(p.locator("dialog .aviso.erro")).to_have_text(
                    "Não é possível remover ou desativar o único administrador ativo.")
                await p.click("dialog >> text=Cancelar")
            await teste("único administrador não pode ser rebaixado", unico_admin)

            tecnico = await Contexto(nav, viewport={"width": 390, "height": 760}, is_mobile=True, has_touch=True).abrir()
            t = tecnico.p

            async def tecnico_sem_usuarios():
                await tecnico.entrar("teo@loja.com")
                await expect(t.locator("h1")).to_have_text("Olá, Téo")
                await expect(t.locator("nav >> text=Usuários")).to_have_count(0)
                await t.goto(BASE + "/usuarios")
                await expect(t).to_have_url(BASE + "/")
            await teste("técnico não vê nem acessa Usuários", tecnico_sem_usuarios)

            async def navegacao_celular():
                caixa = await t.locator("nav").bounding_box()
                assert caixa["y"] + caixa["height"] >= 759, f"navegação não está na base: {caixa}"
                for el in await t.locator("nav a, header button").all():
                    b = await el.bounding_box()
                    assert b["height"] >= 44, f"alvo de toque com {b['height']}px"
                largura = await t.evaluate("document.documentElement.scrollWidth")
                assert largura <= 390, f"rolagem horizontal: {largura}px"
            await teste("celular: navegação na base, toques de 44 px e sem rolagem lateral", navegacao_celular)

            async def sessao_cai_vai_para_login():
                await p.click("li:has-text('Téo') >> text=Editar")
                await p.check("text=Redefinir a senha")
                await p.fill("#u-senha", "nova-senha-do-teo")
                await p.click("dialog button[type=submit]")
                await expect(p.locator(".aviso.ok")).to_contain_text("Alterações de Téo Técnico salvas")
                await t.click("nav >> text=Minha conta")
                await t.fill("#atual", SENHA)
                await t.fill("#nova", "qualquer-outra")
                await t.fill("#confirmacao", "qualquer-outra")
                await t.click("button[type=submit]")
                await expect(t).to_have_url(BASE + "/entrar")
                await expect(t.locator(".aviso")).to_have_text("Sua sessão foi encerrada. Entre novamente.")
            await teste("sessão encerrada pelo administrador leva ao login com aviso", sessao_cai_vai_para_login)

            async def nova_senha_funciona():
                await tecnico.entrar("teo@loja.com", "nova-senha-do-teo")
                await expect(t.locator("h1")).to_have_text("Olá, Téo")
            await teste("técnico entra com a senha redefinida", nova_senha_funciona)

            async def trocar_propria_senha():
                await t.click("nav >> text=Minha conta")
                await t.fill("#atual", "errada")
                await t.fill("#nova", "senha-nova-teo-2")
                await t.fill("#confirmacao", "senha-nova-teo-2")
                await t.click("button[type=submit]")
                await expect(t.locator(".aviso.erro")).to_have_text("A senha atual não confere.")
                await t.fill("#atual", "nova-senha-do-teo")
                await t.fill("#confirmacao", "diferente")
                await t.click("button[type=submit]")
                await expect(t.locator(".aviso.erro")).to_have_text("As senhas novas não são iguais.")
                await t.fill("#confirmacao", "senha-nova-teo-2")
                await t.click("button[type=submit]")
                await expect(t.locator(".aviso.ok")).to_contain_text("Senha alterada")
            await teste("troca da própria senha valida e confirma", trocar_propria_senha)

            visitante = await Contexto(nav, viewport={"width": 1280, "height": 800}).abrir()
            v = visitante.p

            async def protegido_sem_login():
                for rota in ["/", "/usuarios", "/conta"]:
                    await v.goto(BASE + rota)
                    await expect(v).to_have_url(BASE + "/entrar")
            await teste("sem login, qualquer tela leva à entrada", protegido_sem_login)

            async def senha_errada_e_bloqueio():
                for _ in range(5):
                    await visitante.entrar("ana@loja.com", "errada")
                    await expect(v.locator(".aviso.erro")).to_have_text("E-mail ou senha incorretos.")
                await visitante.entrar("ana@loja.com")
                await expect(v.locator(".aviso.erro")).to_contain_text("Acesso bloqueado temporariamente")
                await p.reload()
                await expect(p.locator("li:has-text('Ana Bloqueio')")).to_contain_text("Bloqueado por tentativas")
            await teste("senha errada avisa e 5 erros bloqueiam (visível ao admin)", senha_errada_e_bloqueio)

            async def desbloquear_redefinindo():
                await p.click("li:has-text('Ana') >> text=Editar")
                await p.check("text=Redefinir a senha")
                await p.fill("#u-senha", "senha-nova-ana")
                await p.click("dialog button[type=submit]")
                await expect(p.locator("li:has-text('Ana Bloqueio')")).not_to_contain_text("Bloqueado")
                await visitante.entrar("ana@loja.com", "senha-nova-ana")
                await expect(v.locator("h1")).to_have_text("Olá, Ana")
            await teste("redefinir a senha desbloqueia o usuário", desbloquear_redefinindo)

            async def desativar():
                await p.click("li:has-text('Bia') >> text=Editar")
                await p.uncheck("text=Pode entrar no sistema")
                await p.click("dialog button[type=submit]")
                await expect(p.locator("li:has-text('Bia Balcão')")).to_contain_text("Desativado")
                outro = await Contexto(nav).abrir()
                await outro.entrar("bia@loja.com")
                await expect(outro.p.locator(".aviso.erro")).to_have_text("E-mail ou senha incorretos.")
                await outro.ctx.close()
            await teste("usuário desativado não entra", desativar)

            async def pagina_inexistente():
                await p.goto(BASE + "/endereco/que/nao/existe")
                await expect(p.locator("h1")).to_have_text("Página não encontrada")
                corpo = await p.locator("body").inner_text()
                assert "Not Found" not in corpo, "mensagem em inglês"
            await teste("endereço inexistente mostra aviso em português", pagina_inexistente)

            async def sair():
                await p.goto(BASE + "/")
                await p.click("header >> text=Sair")
                await expect(p).to_have_url(BASE + "/entrar")
                await p.go_back()
                await expect(p).to_have_url(BASE + "/entrar")
            await teste("sair encerra a sessão, inclusive pelo botão voltar", sair)

            async def sem_erros_de_console():
                todos = admin.erros + tecnico.erros + visitante.erros
                # Respostas 4xx esperadas (senha errada etc.) aparecem como erro de rede no console.
                reais = [e for e in todos if "Failed to load resource" not in e]
                assert not reais, f"erros no console: {reais[:3]}"
            await teste("nenhum erro de JavaScript no console", sem_erros_de_console)

            await nav.close()

        # Reinício: dados e sessões sobrevivem.
        parar(proc)
        proc = subir(dados)

        async def persistencia():
            async with async_playwright() as pw:
                nav = await pw.chromium.launch()
                c = await Contexto(nav).abrir()
                await c.entrar("liandro@loja.com")
                await expect(c.p.locator("h1")).to_have_text("Olá, Liandro")
                await c.p.click("nav >> text=Usuários")
                await expect(c.p.locator(".lista li")).to_have_count(4)
                await nav.close()
        await teste("dados continuam após reiniciar o servidor", persistencia)
    finally:
        parar(proc)
        shutil.rmtree(dados, ignore_errors=True)

    falhas = [r for r in resultados if not r[0]]
    print(f"\n{len(resultados) - len(falhas)} de {len(resultados)} testes passaram.")
    sys.exit(1 if falhas else 0)


asyncio.run(principal())
