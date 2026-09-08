---
titulo: SRS — fire
data: 2026-08-18
status: entregue
comando: gtr fire
apelido: jam
pacotes:
  - cmd/fire.go
  - internal/fire
  - internal/riff
commits:
  - db7b6bd
  - f46791a
fonte_externa: whatthecommit.com, por intermédio do gtr riff
---

# SRS — fire

- **Data:** 2026-08-18
- **Feature:** `cmd/fire` + `internal/fire`
- **Status:** **entregue** — commits `db7b6bd` (domínio) e `f46791a` (comando)
- **Fonte externa:** whatthecommit.com, por intermédio do [`gtr riff`](riff.md)

---

## 1. Introdução

### 1.1 Propósito

Especificar o `gtr fire`, o **botão de pânico**: salva tudo que está sujo — rastreado ou não — num commit só e empurra para uma branch nova no remoto, **sem perguntar**. Existe para o momento em que o operador precisa sair agora e não pode arriscar perder trabalho local.

### 1.2 Escopo

**Entregue:** detectar sujeira com `git status --porcelain`; `git add -A` seguido de commit; push para uma branch remota nova, nomeada pelo SHA do commit; mensagem vinda do `gtr riff`, com mensagem fixa de reserva se a rede falhar; nenhuma confirmação pedida; linha de recuperação local impressa ao final.

**Fora de escopo:** não mexe no upstream já configurado da branch atual — só cria destino novo. Não é substituto de `git stash`. Não desfaz nada; quem quer desfazer usa [`gtr undo`](undo.md) ou [`gtr author --reset`](author.md) depois.

### 1.3 O nome

`fire`, inspirado no projeto de piada `git-fire`. Apelido `jam`, de duplo sentido: estar **travado** numa situação e uma **jam session** — combina com o `gtr` ser abreviação de guitarra.

---

## 2. Descrição geral

```text
internal/fire/               domínio — não imprime nada
└── repo.go                  Dirty, Save, Ensure; Save{Head, Branch}

cmd/fire.go                  o comando, sem --format, sem confirmação; riffOrFallback
```

O `fire` é o único comando que combina **dois domínios** numa única operação: `internal/fire` (o commit e o push) e `internal/riff` (a mensagem), injetado a partir do `web.Client` do `cmd`. Os dois domínios continuam sem se conhecer — a composição vive inteira em `cmd/fire.go`.

---

## 3. Requisitos funcionais

### RF-01 — Detectar sujeira

`git status --porcelain` decide se há algo para salvar. Árvore limpa: nada roda, saída `Nada sujo para salvar.`, sem erro.

### RF-02 — Salvar tudo, rastreado ou não

`git add -A` seguido de commit, cobrindo **tudo** que está sujo — arquivo novo, modificado ou só no diretório de trabalho.

### RF-03 — Empurrar para uma branch remota nova

O commit vai para `fire/<SHA curto do commit novo>` no remoto, **nunca** para o upstream já configurado da branch local atual.

### RF-04 — Mensagem via `riff`, com reserva

A mensagem do commit vem do [`gtr riff`](riff.md); se a chamada falhar, usa a mensagem fixa `🔥 fire` e segue — **o pânico não espera a internet**.

### RF-05 — Sem confirmação

O `fire` roda direto, sem `[y/N]`. É a única exceção deliberada, em todo o projeto, à disciplina de sempre confirmar antes de mexer em remoto — **ADR-008**. A justificativa é textual no próprio `--help`.

### RF-06 — Relatar o resultado

Após salvar, imprime a branch destino, a mensagem usada e a linha `Recuperável com: git reset --hard <head anterior>`.

### RF-07 — Declarar a chamada de rede

O `--help` avisa que o comando **faz chamada de rede** para a mensagem — mesma disciplina do `riff`.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **"Sujo" aqui é tudo que `git add -A` pegaria** — rastreado ou não, staged ou não —, o **oposto** do escopo que o `gtr author --reset` usa para a mesma palavra: lá, só interessa mudança em arquivo **já rastreado** (`diff HEAD --quiet`). Os dois comandos usam "sujo" com abrangências diferentes de propósito — ver §5.3. |
| **RN-02** | **O nome da branch remota é `fire/<sha do commit>`, nunca um carimbo de hora.** Não depende do relógio, não colide entre duas chamadas rápidas que gerem commits diferentes, e cada `fire` sempre tem destino remoto próprio. |
| **RN-03** | **O `HEAD` anterior é lido como `HEAD^` depois do commit**, nunca por uma leitura de `HEAD` antes dele. As duas leituras seriam o mesmo comando de git (`rev-parse --short HEAD`) com respostas que têm de ser diferentes — e nada no texto do comando distingue uma chamada da outra. `HEAD^` depois do commit aponta para o mesmo SHA que o `HEAD` apontava antes, resolvendo as duas necessidades com uma chamada só. |
| **RN-04** | **O push tem como alvo uma ref nova (`refs/heads/fire/<sha>`), nunca a branch de upstream já configurada.** A cópia de segurança nunca sobrescreve o que já estava no remoto. |
| **RN-05** | **Falha da chamada de rede nunca vira falha do `fire`.** Cai para a mensagem fixa e segue salvando — a rede é cosmética aqui, nunca crítica. |
| **RN-06** | **O `fire` é a única exceção documentada à regra de sempre confirmar antes de tocar em remoto** (**ADR-008**). Decisão explícita, não descuido: um botão de pânico que pergunta não serve ao próprio propósito. |

### 4.1 Regras herdadas

- **Nenhum comando sai da máquina sem dizer** — `RN-02` da [SRS — a conexão](doctor-online.md), cumprida pela `RF-07`.
- **Toda chamada de rede tem prazo** — `RN-03` da [SRS — a conexão](doctor-online.md), aqui num contexto próprio e aninhado: ver §5.2.

---

## 5. Achados de implementação

### 5.1 Por que `HEAD^` depois do commit, e não `HEAD` antes

Ler `HEAD` antes e depois do commit seria **o mesmo comando de git, duas vezes**, com respostas que precisam divergir — e o fake de testes é indexado pelo texto exato do comando, então duas chamadas idênticas não teriam como responder coisas diferentes de qualquer jeito.

Ler `HEAD^` **depois** do commit resolve isso com uma única chamada, porque aponta exatamente para o SHA que o `HEAD` apontava antes.

### 5.2 O prazo de rede fica isolado dentro do `fire`

O `riffOrFallback` abre seu **próprio** `context.WithTimeout`, com o mesmo `networkDeadline`, aninhado dentro do contexto do comando — uma falha ali, por prazo estourado ou erro de rede, nunca propaga como erro do `fire`: só aciona o fallback.

### 5.3 "Sujo" no `fire` e no `author --reset` não é a mesma pergunta

Os dois comandos perguntam "há algo não commitado?", mas com escopos opostos por desenho: o `author --reset` só avisa sobre arquivo **já rastreado**, porque é isso que o `--hard` de fato descarta sem rastro; o `fire` quer **tudo**, porque o propósito é não perder nada, rastreado ou não.

Nenhum dos dois está errado — são perguntas diferentes que compartilham a palavra, e vale saber disso ao ler os dois comandos esperando o mesmo escopo.

---

## 6. Testes

**100% de statements** em `internal/fire` e em `cmd/fire.go`.

| # | Cenário | Esperado |
|---|---|---|
| 1 | Árvore suja mista (rastreado + novo), riff respondendo | Commit, push, branch e mensagem certas na saída, recuperação impressa — `RF-01` a `RF-04`, `RF-06` |
| 2 | Riff falha | Mensagem fixa `🔥 fire`, sem erro — `RF-04`, `RN-05` |
| 3 | Árvore limpa | Nada roda, mensagem específica, sem `add` — `RF-01` |
| 4 | Fora de um repositório | Erro do `Ensure` |
| 5 | Falha do `status` | Erro propagado |
| 6 | Falha do push, dentro do `Save` | Erro propagado |
| 7 | Argumento posicional sobrando | Erro |
| 8 | `--help` | Menciona a chamada de rede — `RF-07` |
| 9 | `jam` | Roda o mesmo comando, fora da ajuda da raiz — §1.3 |
| 10 | `Dirty` com mudança mista, árvore limpa, e falha do `status` | `true`, `false`, erro — `RN-01` |
| 11 | `Save`: falha do `add`, do `commit`, da leitura do `HEAD` anterior, da leitura do `HEAD` novo, e do `push`, cada uma isolada | Erro propagado em cada etapa |

---

## 7. Rastreabilidade

| Commit | Entrega |
|---|---|
| `db7b6bd` | `internal/fire.Repo`: `Dirty` e `Save` — add, commit, `HEAD^` pós-commit, push para `fire/<sha>` — `RF-01` a `RF-03`, `RN-01` a `RN-04`, §5.1 |
| `f46791a` | `gtr fire`: o comando, sem confirmação, mensagem via `riff` com reserva, apelido `jam` — `RF-04` a `RF-07`, `RN-05`, `RN-06`, §5.2 |

---

## 8. Follow-ups conhecidos

- **Sem `--dry-run`.** Não há como ver o que seria salvo sem de fato commitar e empurrar. Aceito porque contradiz o próprio propósito: um botão de pânico que pede conferência antes deixa de ser botão de pânico.
- **Sem mensagem manual.** Sempre `riff` ou a mensagem fixa de reserva; quem precisa de mensagem específica reescreve depois com [`gtr author`](author.md).
