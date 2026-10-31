<script>
  // Moldura das telas antes do login: nome da loja em destaque e o formulário.
  import Tema from './Tema.svelte';
  let { titulo, subtitulo, children } = $props();
</script>

<div class="tela">
  <div class="topo"><Tema /></div>
  <main>
    <h1>{titulo}</h1>
    {#if subtitulo}<p class="sub">{subtitulo}</p>{/if}
    {@render children()}
  </main>
</div>

<style>
  .tela { min-height: 100dvh; display: grid; grid-template-rows: auto 1fr; }
  .topo { display: flex; justify-content: flex-end; padding: max(1rem, env(safe-area-inset-top)) 1rem 0; }
  main {
    width: 100%;
    max-width: 26rem;
    margin: 0 auto;
    padding: 2rem 1.25rem 3rem;
    align-self: center;
  }
  h1 {
    font-size: clamp(2rem, 7vw, 2.75rem);
    letter-spacing: -0.01em;
    overflow-wrap: anywhere;
    padding-bottom: 0.6rem;
    border-bottom: 4px solid var(--acao);
    display: inline-block;
  }
  .sub { color: var(--texto-2); margin: 1rem 0 0; }
</style>
