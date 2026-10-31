<script>
  import { sessao } from '$lib/sessao.svelte.js';
  import { PERFIS } from '$lib/api.js';
</script>

<h1>Olá, {sessao.usuario.nome.split(' ')[0]}</h1>
<p class="sub">Você entrou como {PERFIS[sessao.usuario.perfil]}.</p>

<section class="painel">
  <h2>Sistema instalado</h2>
  <p>O servidor está funcionando e os dados ficam salvos no ZimaOS.</p>
  {#if sessao.usuario.perfil === 'admin'}
    <p>Comece cadastrando quem vai usar o sistema: cada pessoa da loja tem o próprio login.</p>
    <a class="botao primario" href="/usuarios">Cadastrar usuários</a>
  {/if}
  <p class="versao">Versão {sessao.versao}</p>
</section>

<style>
  h1 { font-size: 1.75rem; }
  .sub { color: var(--texto-2); margin: 0.35rem 0 1.5rem; }
  .painel {
    background: var(--superficie);
    border: 1px solid var(--borda);
    border-left: 4px solid var(--acao);
    border-radius: var(--raio);
    padding: 1.25rem;
    max-width: 38rem;
  }
  .painel h2 { font-size: 1.2rem; margin-bottom: 0.5rem; }
  .painel p { margin: 0 0 1rem; }
  .versao { color: var(--texto-2); font-size: 0.875rem; margin: 1.25rem 0 0 !important; }
</style>
