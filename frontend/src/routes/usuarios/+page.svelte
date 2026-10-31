<script>
  import { onMount } from 'svelte';
  import { api, PERFIS } from '$lib/api.js';
  import { sessao } from '$lib/sessao.svelte.js';

  let usuarios = $state([]);
  let carregando = $state(true);
  let erroLista = $state('');
  let mensagem = $state('');

  // Formulário (novo ou edição) num diálogo.
  let dialogo;
  let editando = $state(null); // null = novo usuário
  let form = $state({ nome: '', email: '', senha: '', perfil: 'balcao', ativo: true });
  let redefinir = $state(false);
  let erroForm = $state('');
  let salvando = $state(false);

  async function carregar() {
    erroLista = '';
    try {
      usuarios = await api('GET', '/usuarios');
    } catch (e) {
      erroLista = e.message;
    } finally {
      carregando = false;
    }
  }
  onMount(carregar);

  function abrirNovo() {
    editando = null;
    form = { nome: '', email: '', senha: '', perfil: 'balcao', ativo: true };
    redefinir = false;
    erroForm = '';
    dialogo.showModal();
  }

  function abrirEdicao(u) {
    editando = u;
    form = { nome: u.nome, email: u.email, senha: '', perfil: u.perfil, ativo: u.ativo };
    redefinir = false;
    erroForm = '';
    dialogo.showModal();
  }

  async function salvar(e) {
    e.preventDefault();
    erroForm = '';
    salvando = true;
    try {
      if (editando) {
        const dados = { nome: form.nome, perfil: form.perfil, ativo: form.ativo };
        if (redefinir) dados.nova_senha = form.senha;
        await api('PUT', `/usuarios/${editando.id}`, dados);
        mensagem = `Alterações de ${form.nome} salvas.`;
      } else {
        await api('POST', '/usuarios', form);
        mensagem = `${form.nome} cadastrado. Informe o e-mail e a senha a essa pessoa.`;
      }
      dialogo.close();
      await carregar();
    } catch (err) {
      erroForm = err.message;
    } finally {
      salvando = false;
    }
  }
</script>

<div class="cabecalho">
  <h1>Usuários</h1>
  <button class="primario" onclick={abrirNovo}>Cadastrar usuário</button>
</div>
<p class="sub">Cada pessoa da loja tem o próprio login. O perfil define o que ela pode fazer.</p>

{#if mensagem}<div class="aviso ok" role="status">{mensagem}</div>{/if}
{#if erroLista}<div class="aviso erro" role="alert">{erroLista}</div>{/if}

{#if carregando}
  <p class="sub">Carregando…</p>
{:else}
  <ul class="lista">
    {#each usuarios as u (u.id)}
      <li class:inativo={!u.ativo}>
        <div class="quem">
          <strong>{u.nome}{u.id === sessao.usuario.id ? ' (você)' : ''}</strong>
          <span class="email">{u.email}</span>
        </div>
        <div class="etiquetas">
          <span class="perfil perfil-{u.perfil}">{PERFIS[u.perfil]}</span>
          {#if !u.ativo}<span class="estado">Desativado</span>{/if}
          {#if u.bloqueado}<span class="estado alerta">Bloqueado por tentativas</span>{/if}
        </div>
        <button class="secundario" onclick={() => abrirEdicao(u)} aria-label="Editar {u.nome}">Editar</button>
      </li>
    {/each}
  </ul>
{/if}

<details class="perfis">
  <summary>O que cada perfil pode fazer</summary>
  <dl>
    <dt>Administrador</dt><dd>Tudo, incluindo configurações, usuários, backups e relatórios financeiros.</dd>
    <dt>Balcão</dt><dd>Abrir, entregar e cancelar O.S., cadastrar clientes, vender, receber pagamentos e operar o caixa.</dd>
    <dt>Técnico</dt><dd>Ver e atualizar as próprias O.S., laudos, peças e fotos. Não vê o financeiro.</dd>
  </dl>
</details>

<dialog bind:this={dialogo} aria-labelledby="titulo-dialogo">
  <form onsubmit={salvar} novalidate>
    <h2 id="titulo-dialogo">{editando ? `Editar ${editando.nome}` : 'Cadastrar usuário'}</h2>
    {#if erroForm}<div class="aviso erro" role="alert">{erroForm}</div>{/if}
    <div class="campo">
      <label for="u-nome">Nome</label>
      <input id="u-nome" required bind:value={form.nome} />
    </div>
    {#if !editando}
      <div class="campo">
        <label for="u-email">E-mail</label>
        <input id="u-email" type="email" inputmode="email" autocomplete="off" required bind:value={form.email} />
      </div>
    {/if}
    <div class="campo">
      <label for="u-perfil">Perfil</label>
      <select id="u-perfil" bind:value={form.perfil}>
        <option value="balcao">Balcão</option>
        <option value="tecnico">Técnico</option>
        <option value="admin">Administrador</option>
      </select>
    </div>
    {#if editando}
      <label class="opcao"><input type="checkbox" bind:checked={form.ativo} /> Pode entrar no sistema</label>
      <label class="opcao"><input type="checkbox" bind:checked={redefinir} /> Redefinir a senha</label>
    {/if}
    {#if !editando || redefinir}
      <div class="campo">
        <label for="u-senha">{editando ? 'Nova senha' : 'Senha inicial'}</label>
        <input id="u-senha" type="text" autocomplete="off" required bind:value={form.senha} />
        <span class="dica">Pelo menos 8 caracteres. A pessoa pode trocar depois em Minha conta.</span>
      </div>
    {/if}
    <div class="botoes">
      <button type="button" class="secundario" onclick={() => dialogo.close()}>Cancelar</button>
      <button type="submit" class="primario" disabled={salvando}>{salvando ? 'Salvando…' : editando ? 'Salvar alterações' : 'Cadastrar'}</button>
    </div>
  </form>
</dialog>

<style>
  .cabecalho { display: flex; flex-wrap: wrap; gap: 1rem; align-items: center; justify-content: space-between; }
  h1 { font-size: 1.75rem; }
  .sub { color: var(--texto-2); margin: 0.35rem 0 1.25rem; }
  .aviso { margin-bottom: 1rem; }
  .lista { list-style: none; margin: 0; padding: 0; border: 1px solid var(--borda); border-radius: var(--raio); background: var(--superficie); }
  .lista li {
    display: grid;
    grid-template-columns: 1fr auto auto;
    gap: 0.75rem 1rem;
    align-items: center;
    padding: 0.85rem 1rem;
  }
  .lista li + li { border-top: 1px solid var(--borda); }
  .inativo .quem { opacity: 0.6; }
  .quem { display: grid; min-width: 0; }
  .email { color: var(--texto-2); font-size: 0.9rem; overflow-wrap: anywhere; }
  .etiquetas { display: flex; flex-wrap: wrap; gap: 0.4rem; justify-content: flex-end; }
  .perfil, .estado { font-size: 0.85rem; padding: 0.15rem 0.6rem; border-radius: 999px; border: 1px solid var(--borda); white-space: nowrap; }
  .perfil-admin { border-color: var(--acao); color: var(--acao-forte); }
  .estado { color: var(--texto-2); }
  .estado.alerta { color: var(--erro); border-color: var(--erro); }
  @media (max-width: 640px) {
    .lista li { grid-template-columns: 1fr auto; }
    .etiquetas { grid-column: 1; grid-row: 2; justify-content: flex-start; }
    .lista li button { grid-row: 1 / span 2; grid-column: 2; }
  }
  .perfis { margin-top: 1.5rem; color: var(--texto-2); }
  .perfis summary { cursor: pointer; padding: 0.6rem 0; font-weight: 700; }
  .perfis dt { color: var(--texto); font-weight: 700; margin-top: 0.75rem; }
  .perfis dd { margin: 0.2rem 0 0; }

  dialog {
    width: min(30rem, calc(100vw - 2rem));
    border: 1px solid var(--borda);
    border-radius: var(--raio);
    background: var(--superficie);
    color: var(--texto);
    padding: 1.5rem;
  }
  dialog::backdrop { background: rgb(0 0 0 / 0.55); }
  dialog form { display: grid; gap: 1rem; }
  dialog h2 { font-size: 1.3rem; }
  .opcao { display: flex; align-items: center; gap: 0.6rem; min-height: 44px; cursor: pointer; }
  .opcao input { width: 1.25rem; min-height: 1.25rem; accent-color: var(--acao); }
  .botoes { display: flex; gap: 0.75rem; justify-content: flex-end; flex-wrap: wrap; margin-top: 0.5rem; }
</style>
