<script>
  import { goto } from '$app/navigation';
  import { api } from '$lib/api.js';
  import { sessao } from '$lib/sessao.svelte.js';
  import Entrada from '$lib/Entrada.svelte';

  let email = $state('');
  let senha = $state('');
  let erro = $state('');
  let enviando = $state(false);

  async function entrar(e) {
    e.preventDefault();
    erro = '';
    sessao.aviso = '';
    enviando = true;
    try {
      sessao.usuario = await api('POST', '/auth/entrar', { email, senha });
      goto('/', { replaceState: true });
    } catch (err) {
      erro = err.message;
      senha = '';
    } finally {
      enviando = false;
    }
  }
</script>

<Entrada titulo={sessao.nomeLoja} subtitulo="Entre com seu e-mail e senha.">
  <form onsubmit={entrar} novalidate>
    {#if sessao.aviso && !erro}<div class="aviso erro" role="status">{sessao.aviso}</div>{/if}
    {#if erro}<div class="aviso erro" role="alert">{erro}</div>{/if}
    <div class="campo">
      <label for="email">E-mail</label>
      <input id="email" type="email" autocomplete="username" inputmode="email" required bind:value={email} />
    </div>
    <div class="campo">
      <label for="senha">Senha</label>
      <input id="senha" type="password" autocomplete="current-password" required bind:value={senha} />
    </div>
    <button class="primario" type="submit" disabled={enviando}>{enviando ? 'Entrando…' : 'Entrar'}</button>
    <p class="ajuda">Esqueceu a senha? Peça ao administrador para redefinir.</p>
  </form>
</Entrada>

<style>
  form { display: grid; gap: 1.1rem; margin-top: 2rem; }
  .ajuda { color: var(--texto-2); font-size: 0.9rem; margin: 0; }
</style>
