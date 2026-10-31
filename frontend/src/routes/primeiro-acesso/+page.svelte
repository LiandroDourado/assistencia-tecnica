<script>
  import { goto } from '$app/navigation';
  import { api } from '$lib/api.js';
  import { sessao } from '$lib/sessao.svelte.js';
  import Entrada from '$lib/Entrada.svelte';

  let nomeLoja = $state('');
  let nome = $state('');
  let email = $state('');
  let senha = $state('');
  let confirmacao = $state('');
  let erro = $state('');
  let enviando = $state(false);

  async function criar(e) {
    e.preventDefault();
    erro = '';
    if (senha !== confirmacao) {
      erro = 'As senhas não são iguais.';
      return;
    }
    enviando = true;
    try {
      sessao.usuario = await api('POST', '/primeiro-acesso', { nome_loja: nomeLoja, nome, email, senha });
      sessao.configurado = true;
      if (nomeLoja.trim()) sessao.nomeLoja = nomeLoja.trim();
      goto('/', { replaceState: true });
    } catch (err) {
      erro = err.message;
    } finally {
      enviando = false;
    }
  }
</script>

<Entrada titulo="Bem-vindo" subtitulo="Primeiro acesso: cadastre a loja e o administrador do sistema. O administrador cadastra os demais usuários depois.">
  <form onsubmit={criar} novalidate>
    {#if erro}<div class="aviso erro" role="alert">{erro}</div>{/if}
    <div class="campo">
      <label for="loja">Nome da loja</label>
      <input id="loja" autocomplete="organization" bind:value={nomeLoja} placeholder="Ex.: Conserta Já Celulares" />
    </div>
    <div class="campo">
      <label for="nome">Seu nome</label>
      <input id="nome" autocomplete="name" required bind:value={nome} />
    </div>
    <div class="campo">
      <label for="email">E-mail</label>
      <input id="email" type="email" autocomplete="username" inputmode="email" required bind:value={email} />
      <span class="dica">É com ele que você vai entrar no sistema.</span>
    </div>
    <div class="campo">
      <label for="senha">Senha</label>
      <input id="senha" type="password" autocomplete="new-password" required bind:value={senha} aria-describedby="dica-senha" />
      <span class="dica" id="dica-senha">Pelo menos 8 caracteres.</span>
    </div>
    <div class="campo">
      <label for="confirmacao">Repita a senha</label>
      <input id="confirmacao" type="password" autocomplete="new-password" required bind:value={confirmacao} />
    </div>
    <button class="primario" type="submit" disabled={enviando}>{enviando ? 'Criando…' : 'Criar administrador e entrar'}</button>
  </form>
</Entrada>

<style>
  form { display: grid; gap: 1.1rem; margin-top: 2rem; }
</style>
