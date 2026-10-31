// Cliente da API. Erros chegam do servidor já em português.

export class ErroApi extends Error {
  constructor(status, codigo, mensagem) {
    super(mensagem);
    this.status = status;
    this.codigo = codigo;
  }
}

// Chamado quando a sessão deixa de valer no meio do uso (expirou, a senha foi
// redefinida, o usuário foi desativado). Quem registra é sessao.svelte.js.
let aoExpirarSessao = () => {};
export function quandoSessaoExpirar(f) {
  aoExpirarSessao = f;
}

// Rotas em que 401 é uma resposta normal, não sessão expirada.
const SEM_SESSAO = ['/auth/entrar', '/auth/eu', '/primeiro-acesso'];

export async function api(metodo, caminho, corpo) {
  let resposta;
  try {
    resposta = await fetch('/api' + caminho, {
      method: metodo,
      headers: corpo === undefined ? {} : { 'Content-Type': 'application/json' },
      body: corpo === undefined ? undefined : JSON.stringify(corpo),
      credentials: 'same-origin'
    });
  } catch {
    throw new ErroApi(0, 'SEM_CONEXAO', 'Sem conexão com o servidor. Confira a rede da loja e tente de novo.');
  }
  let dados = null;
  try {
    dados = await resposta.json();
  } catch {
    /* resposta sem corpo JSON */
  }
  if (!resposta.ok) {
    if (resposta.status === 401 && !SEM_SESSAO.includes(caminho)) aoExpirarSessao();
    throw new ErroApi(resposta.status, dados?.codigo ?? 'ERRO', dados?.erro ?? 'Não foi possível concluir a operação.');
  }
  return dados;
}

export const PERFIS = {
  admin: 'Administrador',
  balcao: 'Balcão',
  tecnico: 'Técnico'
};
