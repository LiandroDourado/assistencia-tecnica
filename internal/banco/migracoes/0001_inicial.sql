-- Entrega 1: usuários, sessões, auditoria e configurações.
-- Convenções: datas/horas em TEXT ISO-8601 UTC; nenhuma migração aplicada é
-- alterada depois — mudanças entram sempre num arquivo novo.

CREATE TABLE usuarios (
    id                INTEGER PRIMARY KEY,
    nome              TEXT    NOT NULL CHECK (length(trim(nome)) > 0),
    email             TEXT    NOT NULL UNIQUE COLLATE NOCASE,
    senha_hash        TEXT    NOT NULL,
    perfil            TEXT    NOT NULL CHECK (perfil IN ('admin', 'balcao', 'tecnico')),
    ativo             INTEGER NOT NULL DEFAULT 1 CHECK (ativo IN (0, 1)),
    tentativas_falhas INTEGER NOT NULL DEFAULT 0,
    bloqueado_ate     TEXT,
    criado_em         TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE sessoes (
    token_hash TEXT    PRIMARY KEY,
    usuario_id INTEGER NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    expira_em  TEXT    NOT NULL,
    criado_em  TEXT    NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_sessoes_usuario ON sessoes(usuario_id);
CREATE INDEX idx_sessoes_expira ON sessoes(expira_em);

-- Log de auditoria: não guarda senhas nem dados pessoais de clientes (LGPD).
CREATE TABLE auditoria (
    id          INTEGER PRIMARY KEY,
    usuario_id  INTEGER REFERENCES usuarios(id),
    entidade    TEXT    NOT NULL,
    entidade_id INTEGER,
    acao        TEXT    NOT NULL,
    antes       TEXT,
    depois      TEXT,
    criado_em   TEXT    NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_auditoria_entidade ON auditoria(entidade, entidade_id);

CREATE TRIGGER trg_auditoria_sem_update BEFORE UPDATE ON auditoria
BEGIN SELECT RAISE(ABORT, 'AUDITORIA_IMUTAVEL'); END;
CREATE TRIGGER trg_auditoria_sem_delete BEFORE DELETE ON auditoria
BEGIN SELECT RAISE(ABORT, 'AUDITORIA_IMUTAVEL'); END;

CREATE TABLE configuracoes (
    chave TEXT PRIMARY KEY,
    valor TEXT NOT NULL
);

INSERT INTO configuracoes (chave, valor) VALUES
    ('empresa.nome', 'Minha Assistência'),
    ('seguranca.max_tentativas', '5'),
    ('seguranca.bloqueio_minutos', '15'),
    ('seguranca.sessao_horas', '12');
