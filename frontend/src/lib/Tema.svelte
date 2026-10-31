<script>
  // Alterna entre tema claro e escuro. A escolha fica salva neste navegador.
  let escuro = $state(true);

  function atual() {
    return document.documentElement.dataset.tema !== 'claro';
  }

  $effect(() => {
    escuro = atual();
  });

  function alternar() {
    escuro = !escuro;
    const t = escuro ? 'escuro' : 'claro';
    document.documentElement.dataset.tema = t;
    document.querySelector('meta[name="theme-color"]')?.setAttribute('content', escuro ? '#1a1f26' : '#f2f4f6');
    try { localStorage.setItem('tema', t); } catch {}
  }
</script>

<button type="button" class="secundario tema" onclick={alternar} aria-label={escuro ? 'Usar tema claro' : 'Usar tema escuro'} title={escuro ? 'Usar tema claro' : 'Usar tema escuro'}>
  {#if escuro}
    <svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true"><circle cx="12" cy="12" r="4.5" fill="none" stroke="currentColor" stroke-width="2"/><path d="M12 2v2.5M12 19.5V22M2 12h2.5M19.5 12H22M4.9 4.9l1.8 1.8M17.3 17.3l1.8 1.8M4.9 19.1l1.8-1.8M17.3 6.7l1.8-1.8" stroke="currentColor" stroke-width="2" stroke-linecap="round"/></svg>
  {:else}
    <svg viewBox="0 0 24 24" width="20" height="20" aria-hidden="true"><path d="M20 14.5A8.5 8.5 0 0 1 9.5 4a8.5 8.5 0 1 0 10.5 10.5z" fill="none" stroke="currentColor" stroke-width="2" stroke-linejoin="round"/></svg>
  {/if}
</button>

<style>
  .tema { padding: 0; width: 44px; }
</style>
