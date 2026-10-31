<script>
  import { api, PERFIS } from '$lib/api.js';
  import { sessao } from '$lib/sessao.svelte.js';

  let atual = $state('');
  let nova = $state('');
  let confirmacao = $state('');
  let erro = $state('');
  let ok = $state('');
  let enviando = $state(false);

  async function trocar(e) {
    e.preventDefault();
    erro = '';
    ok = '';
    if (nova !== confirmacao) {
      erro = 'As senhas novas não são iguais.';
      return;
    }
    enviando = true;
    try {
      await api('PUT', '/auth/senha', { senha_atual: atual, nova_senha: nova });
      ok = 'Senha alterada. Os outros aparelhos em que você estava conectado precisarão entrar de novo.';
      atual = nova = confirmacao = '';
    } catch (err) {
      erro = err.message;
    } finally {
      enviando = false;
    }
  }
</script>

<h1>Minha conta</h1>
<p class="sub">{sessao.usuario.nome} · {sessao.usuario.email} · {PERFIS[sessao.usuario.perfil]}</p>

<form onsubmit={trocar} novalidate>
  <h2>Trocar senha</h2>
  {#if erro}<div class="aviso erro" role="alert">{erro}</div>{/if}
  {#if ok}<div class="aviso ok" role="status">{ok}</div>{/if}
  <div class="campo">
    <label for="atual">Senha atual</label>
    <input id="atual" type="password" autocomplete="current-password" required bind:value={atual} />
  </div>
  <div class="campo">
    <label for="nova">Nova senha</label>
    <input id="nova" type="password" autocomplete="new-password" required bind:value={nova} />
    <span class="dica">Pelo menos 8 caracteres.</span>
  </div>
  <div class="campo">
    <label for="confirmacao">Repita a nova senha</label>
    <input id="confirmacao" type="password" autocomplete="new-password" required bind:value={confirmacao} />
  </div>
  <button class="primario" type="submit" disabled={enviando}>{enviando ? 'Salvando…' : 'Trocar senha'}</button>
</form>

<style>
  h1 { font-size: 1.75rem; }
  .sub { color: var(--texto-2); margin: 0.35rem 0 1.5rem; overflow-wrap: anywhere; }
  form {
    display: grid;
    gap: 1rem;
    max-width: 26rem;
    background: var(--superficie);
    border: 1px solid var(--borda);
    border-radius: var(--raio);
    padding: 1.25rem;
  }
  h2 { font-size: 1.2rem; }
  button { justify-self: start; }
</style>
