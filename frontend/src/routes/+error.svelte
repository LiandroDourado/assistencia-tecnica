<script>
  // Erros inesperados da interface, sempre em português.
  import { page } from '$app/state';
</script>

<main>
  <h1>{page.status === 404 ? 'Página não encontrada' : 'Algo deu errado'}</h1>
  <p>
    {page.status === 404
      ? 'Este endereço não existe no sistema. Confira o link ou volte ao início.'
      : 'A tela não pôde ser carregada. Recarregue a página; se o problema continuar, avise o administrador.'}
  </p>
  <a class="botao primario" href="/">Ir para o início</a>
</main>

<style>
  main { max-width: 32rem; margin: 0 auto; padding: 3rem 1.25rem; }
  h1 { font-size: 1.75rem; margin-bottom: 0.5rem; }
  p { color: var(--texto-2); margin: 0 0 1.25rem; }
</style>
