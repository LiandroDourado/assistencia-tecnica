<script>
  import '../app.css';
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/state';
  import { sessao, carregarSessao, sair } from '$lib/sessao.svelte.js';
  import Tema from '$lib/Tema.svelte';

  let { children } = $props();
  let erroInicial = $state('');
  const rotasPublicas = ['/entrar', '/primeiro-acesso'];
  const rotasAdmin = ['/usuarios'];

  async function iniciar() {
    erroInicial = '';
    try {
      await carregarSessao();
    } catch (e) {
      erroInicial = e.message;
    }
  }
  onMount(iniciar);

  // Leva cada pessoa para onde ela deve estar: configuração, entrada ou sistema.
  $effect(() => {
    if (!sessao.carregado) return;
    const rota = page.url.pathname;
    if (!sessao.configurado) {
      if (rota !== '/primeiro-acesso') goto('/primeiro-acesso', { replaceState: true });
    } else if (!sessao.usuario) {
      if (rota !== '/entrar') goto('/entrar', { replaceState: true });
    } else if (rotasPublicas.includes(rota)) {
      goto('/', { replaceState: true });
    } else if (rotasAdmin.includes(rota) && sessao.usuario.perfil !== 'admin') {
      goto('/', { replaceState: true });
    }
  });

  const publica = $derived(rotasPublicas.includes(page.url.pathname));
  const links = $derived([
    { href: '/', rotulo: 'Início' },
    ...(sessao.usuario?.perfil === 'admin' ? [{ href: '/usuarios', rotulo: 'Usuários' }] : []),
    { href: '/conta', rotulo: 'Minha conta' }
  ]);

  async function botaoSair() {
    await sair();
    goto('/entrar', { replaceState: true });
  }
</script>

<svelte:head>
  <title>{sessao.nomeLoja}</title>
</svelte:head>

{#if erroInicial}
  <main class="centro">
    <div class="aviso erro" role="alert">{erroInicial}</div>
    <button class="primario" onclick={iniciar}>Tentar de novo</button>
  </main>
{:else if !sessao.carregado}
  <main class="centro" aria-busy="true"><p class="carregando">Carregando…</p></main>
{:else if publica}
  {@render children()}
{:else if sessao.usuario && !(rotasAdmin.includes(page.url.pathname) && sessao.usuario.perfil !== 'admin')}
  <div class="app">
    <header>
      <a class="loja" href="/">{sessao.nomeLoja}</a>
      <nav aria-label="Principal">
        {#each links as l}
          <a href={l.href} aria-current={page.url.pathname === l.href ? 'page' : undefined}>{l.rotulo}</a>
        {/each}
      </nav>
      <div class="acoes">
        <Tema />
        <button class="secundario" onclick={botaoSair}>Sair</button>
      </div>
    </header>
    <main class="conteudo">
      {@render children()}
    </main>
  </div>
{/if}

<style>
  .centro {
    min-height: 100dvh;
    display: grid;
    place-content: center;
    gap: 1rem;
    padding: 1.5rem;
  }
  .carregando { color: var(--texto-2); }
  .app { min-height: 100dvh; }
  header {
    position: sticky;
    top: 0;
    z-index: 10;
    display: flex;
    align-items: center;
    gap: 1.5rem;
    padding: 0.5rem max(1rem, env(safe-area-inset-left));
    padding-top: max(0.5rem, env(safe-area-inset-top));
    background: var(--superficie);
    border-bottom: 1px solid var(--borda);
  }
  .loja {
    font-weight: 700;
    font-size: 1.15rem;
    color: var(--texto);
    text-decoration: none;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  nav { display: flex; gap: 0.25rem; flex: 1; }
  nav a {
    min-height: 44px;
    display: inline-flex;
    align-items: center;
    padding: 0 0.85rem;
    border-radius: var(--raio);
    color: var(--texto-2);
    text-decoration: none;
    font-weight: 700;
  }
  nav a:hover { color: var(--texto); background: var(--superficie-2); }
  nav a[aria-current='page'] { color: var(--texto); background: var(--superficie-2); box-shadow: inset 0 -3px 0 var(--acao); }
  .acoes { display: flex; gap: 0.5rem; margin-left: auto; }
  .conteudo { max-width: 960px; margin: 0 auto; padding: 1.5rem 1rem 6rem; }

  /* No celular, a navegação desce para a base da tela, ao alcance do polegar. */
  @media (max-width: 640px) {
    header { gap: 0.75rem; }
    nav {
      position: fixed;
      left: 0;
      right: 0;
      bottom: 0;
      justify-content: space-around;
      padding: 0.35rem 0.5rem max(0.35rem, env(safe-area-inset-bottom));
      background: var(--superficie);
      border-top: 1px solid var(--borda);
    }
    nav a { flex: 1; justify-content: center; padding: 0 0.25rem; font-size: 0.95rem; }
    nav a[aria-current='page'] { box-shadow: inset 0 3px 0 var(--acao); }
  }
</style>
