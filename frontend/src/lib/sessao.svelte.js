// Estado compartilhado da sessão: loja, usuário logado e se o sistema já foi configurado.
import { api, quandoSessaoExpirar } from './api.js';

export const sessao = $state({
  carregado: false,
  configurado: true,
  nomeLoja: 'Assistência Técnica',
  versao: '',
  usuario: null,
  aviso: '' // mostrado na tela de entrada (ex.: sessão encerrada)
});

quandoSessaoExpirar(() => {
  if (sessao.usuario) {
    sessao.usuario = null;
    sessao.aviso = 'Sua sessão foi encerrada. Entre novamente.';
  }
});

export async function carregarSessao() {
  const estado = await api('GET', '/estado');
  sessao.configurado = estado.configurado;
  sessao.nomeLoja = estado.nome_loja;
  sessao.versao = estado.versao;
  sessao.usuario = null;
  if (estado.configurado) {
    try {
      sessao.usuario = await api('GET', '/auth/eu');
    } catch (e) {
      if (e.status !== 401) throw e;
    }
  }
  sessao.carregado = true;
}

export async function sair() {
  try {
    await api('POST', '/auth/sair');
  } finally {
    sessao.usuario = null;
  }
}
