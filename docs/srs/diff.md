---
titulo: SRS — export de diff
data: 2026-08-07
status: export e apply entregues • seleção interativa fora
comando: gtr diff export • gtr diff apply
pacotes:
  - cmd/diff.go
  - internal/diff
  - internal/git
commits:
  - 6a4cbe8
  - 43c7c98
  - 3d7a876
  - 721f743
  - fc6f118
  - b5c5a92
fonte_externa: nenhuma
---

# SRS — export de diff

- **Data:** 2026-08-07
- **Feature:** `cmd/diff` + `internal/diff`
- **Status:** **entregue** — o `export` não interativo e o `gtr diff apply`. A **seleção interativa (`RF-03`) continua de fora**, dependendo do contrato `Selector`
- **Fonte externa:** nenhuma — só o `git` local

---

## 1. Introdução

### 1.1 Propósito

Especificar `gtr diff export`, que empacota o estado **não commitado** da árvore num patch aplicável, completo e verificado — para mandar a outra pessoa ou guardar fora do repositório.

O problema não é que o git não saiba fazer. É que **o caminho óbvio corrompe em silêncio e o caminho certo é uma incantação que ninguém decora.** `git diff > x.patch` ignora arquivos untracked e reduz binário a `Binary files differ`; o resultado parece uma cópia e não é. Mesma natureza do defeito do `HEAD` destacado na [SRS — branches](branches.md): o comando intuitivo mente, e a correção é compor as peças certas.

### 1.2 Escopo

- Export do estado não commitado da árvore como patch aplicável.
- Inclusão de arquivos **untracked** por padrão, sem tocar no índice do usuário.
- Arquivos **ignorados** apenas sob flag explícita.
- Verificação do patch com `git apply --check` **antes** de entregar.
- Registro da base sobre a qual o patch se aplica.
- `gtr diff apply` — o par do export, embrulho fino sobre o `git apply`.
- Seleção interativa de quais arquivos entram (`-i`), pelo contrato `Selector` da **ADR-002** — **especificada, não implementada**.

**Fora de escopo:**

- **Exportar de uma entrada de stash** — fica para a especificação da feature de stash, ainda não escrita.

**Sobre o `apply` valer a pena:** a régua não é "o git já faz isso numa linha", e sim se o caminho direto é seguro e alguém decora. É por isso que o `export` compõe várias peças — `git diff > x.patch` corrompe em silêncio — e o `apply` só embrulha: o `git apply` já decide certo sozinho, já é atômico, e por isso o embrulho não repete o `--check` que o export faz.

**Esta feature não é destrutiva.** Não cria ref, não escreve no banco de objetos, não altera índice nem árvore. Por construção fica **fora da ADR-008** — e isso é o que permite entregá-la sem nenhuma das três guardas.

### 1.3 Definições

| Termo | Significado |
|---|---|
| **Patch aplicável** | Arquivo que `git apply` reaplica reproduzindo o estado de origem — inclusive arquivos novos e conteúdo binário. Distinto de patch **legível**, que serve para revisão humana e pode omitir binário. |
| **Intent-to-add** (`git add -N`) | Registra o caminho no índice **sem conteúdo**. Efeito útil: o `git diff` passa a enxergar o arquivo como novo e emite o conteúdo inteiro. |
| **Índice temporário** | Índice alternativo apontado por `GIT_INDEX_FILE`. Permite usar `add -N` sem encostar no índice real do usuário. |
| **Tracked / untracked / ignorado** | Três camadas, não duas. Rastreado pelo git; conhecido mas nunca adicionado; casado por `.gitignore`. O `git diff` só vê a primeira. |
| **Base do patch** | SHA do `HEAD` no momento do export — o commit sobre o qual o patch foi tirado e sobre o qual ele reaplica limpo. |

---

## 2. Descrição geral

### 2.1 Posição na arquitetura

```text
internal/git/                execução de git — Run, RunWithInput, RunWithEnv

internal/diff/               domínio — não imprime nada
├── change.go                Change: caminho + camada (tracked/untracked)
├── patch.go                 Patch: conteúdo + base
├── runner.go                a interface maior que este pacote precisa
└── repo.go                  Repo: Ensure, Changes, Export, Verify, Apply

internal/ui/                 o contrato Selector
cmd/diff.go                  front de texto
```

**Esta feature é a segunda pressão registrada sobre o `git.Runner`.** O contrato era `Run(args ...string)`. A [SRS — equivalência](equivalencia.md) já registrou que ele não alimenta `stdin`, o que empurrou a detecção para `commit-tree` em vez de `patch-id`. Aqui a necessidade é outra: **variável de ambiente**, para o `GIT_INDEX_FILE`.

Duas features independentes pedindo a mesma mudança é o sinal que a **ADR-003** chama de melhor evidência de que ela é a certa. A forma ficou a mesma nos dois casos — `Runner` intocado, capacidade nas implementações, interface declarada por quem consome —, e o `RunWithEnv` entrou junto com esta implementação, não antes: sem consumidor nem cenário de teste seria statement que nenhum teste alcança, num projeto que segura 100%.

Nada disso fere a regra de rodar sem shell: a variável vai no `exec.Cmd`, e não há `sh -c`.

### 2.2 Fluxo

1. Verifica que o diretório atual é um repositório git.
2. Resolve a base — `rev-parse HEAD`.
3. Levanta os candidatos: tracked modificados e untracked, **excluindo ignorados** salvo flag.
4. Cria um índice temporário e roda `add -N -f` nele, o que faz o `git diff HEAD` enxergar **todo caminho ausente do HEAD** — untracked, ignorado ou staged — como arquivo criado.
5. Gera o patch com `diff --binary`, restrito aos caminhos selecionados.
6. **Verifica** com `git apply --check --cached`, contra um segundo índice também novo.
7. Só então escreve no `stdout`.

---

## 3. Requisitos funcionais

### RF-01 — Exportar o estado não commitado

`gtr diff export` escreve no `stdout` um patch que reproduz as alterações não commitadas da árvore. O patch inclui conteúdo binário (`--binary`), não só a menção de que o binário mudou.

Sem alteração nenhuma, o comando **não emite patch vazio**: informa que não há o que exportar e sai com 0.

### RF-02 — Incluir untracked sem tocar no índice

Arquivos untracked entram no patch como arquivos novos, com conteúdo completo. A inclusão usa `add -N` num **índice temporário** (`GIT_INDEX_FILE`); o índice real do usuário termina byte a byte igual ao que era.

Arquivos **ignorados** ficam de fora, salvo `--include-ignored`.

### RF-03 — Selecionar arquivos interativamente — não implementado

`gtr diff export -i` lista os candidatos e exporta **apenas os marcados**. Sem `-i`, todos entram.

A seleção usa o contrato `Selector` da **ADR-002**. Este é o **segundo caso de uso** do contrato, e o que revelou que a assinatura esboçada — amarrada a `[]branch.Branch` — não generaliza: arquivo não é branch. Ver §8.

### RF-04 — Registrar a base

O patch carrega, em comentário no cabeçalho, o SHA do `HEAD` de onde foi tirado. Patch sem contexto de base é frágil: quem recebe não sabe onde encostar, e o `git apply` falha com conflito sem explicar a causa real.

### RF-05 — Verificar antes de entregar

Antes de escrever a saída, o patch gerado passa por `git apply --check`. Se não aplicar, o comando **falha com código 1 e não emite patch nenhum**.

Esta é a única garantia da feature que protege **o destinatário**, não quem rodou. O git nunca responde "este patch aplica?" — sem isso, a resposta chega quando o outro lado reclama.

### RF-06 — Falhas com código de saída

Fora de repositório git, com falha na geração ou na verificação, o comando escreve no `stderr` e sai com 1. Nada vai para o `stdout` nesses casos.

### RF-07 — Aplicar o patch de volta

`gtr diff apply [<caminho>]` aplica um patch vindo de um arquivo ou do `stdin`. É embrulho fino: sem `--index`, sem `--cached`, e sem verificação prévia própria — o `git apply` já é atômico e já decide certo sozinho.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **Nem o índice nem a árvore são tocados.** O `add -N` acontece exclusivamente num índice temporário, descartado ao fim. Nenhum arquivo é criado, movido ou removido na árvore. Consequência observável: rodar o export duas vezes seguidas produz saída idêntica, e `git status` antes e depois é igual. **É esta regra que mantém a feature fora da ADR-008.** |
| **RN-02** | **Patch entregue é patch verificado.** A saída só é escrita depois de `git apply --check` passar. **Gerar sem erro não é prova** — o `git diff` retorna 0 tranquilamente enquanto omite untracked e reduz binário. Mesma distância entre "o comando funcionou" e "o resultado presta" que a cláusula 1 da **ADR-008** cobra do lado destrutivo. |
| **RN-03** | **Ignorado nunca entra por padrão.** Varrer o `.gitignore` traria `node_modules` e artefatos de build, gerando patch inútil de centenas de megabytes. A flag existe porque o caso legítimo existe — mandar um `.env` de exemplo —, mas o padrão é o seguro. Mesma distinção que o git faz entre `stash -u` e `stash -a`. |

### 4.1 Regras herdadas

- **A leitura vem de formato contratual** — `RN-05` da [SRS — branches](branches.md). Candidatos saem de `git status --porcelain=v2` e `git ls-files`, nunca da listagem humana. Ver §6.1: a escolha da versão do formato foi cobrada na prática.
- **Comandos git rodam sem shell** — `RN-06` da [SRS — branches](branches.md), inclusive os que levam `GIT_INDEX_FILE`.
- **Patch no `stdout`, mensagem no `stderr`** — `RN-09` da [SRS — branches](branches.md). É o que torna `gtr diff export > mudancas.patch` e `gtr diff export | pbcopy` possíveis sem que aviso nenhum contamine o arquivo.
- **O domínio não escreve na tela nem lê do teclado** — `RN-11` da [SRS — branches](branches.md). `internal/diff` devolve `[]Change` e `Patch`; a camada de cada arquivo é enum, com o rótulo escolhido em `internal/ui`.

---

## 5. Interface

```console
$ gtr diff export > mudancas.patch
3 arquivos exportados, 1 novo. Patch verificado, aplica sobre c0c73d6.

$ gtr diff apply mudancas.patch
Patch aplicado.
```

| Flag de `export` | Padrão | Efeito |
|---|---|---|
| `--include-ignored` | `false` | Inclui arquivos casados pelo `.gitignore` |

A linha de resumo vai para o `stderr`, por isso aparece na tela mesmo com a saída redirecionada.

---

## 6. Testes

### 6.1 O bug que a versão do `--porcelain` cobrou

A primeira leitura de candidatos usava `git status --porcelain -z`, a v1. Toda saída do `git.Runner` passa por `TrimSpace`, e na v1 uma entrada não staged **abre com espaço** (`" M arquivo"`) — quando ela era o primeiro registro da saída inteira, a linha chegava cortada de um caractere.

Trocado para `--porcelain=v2`, em que o tipo de cada linha (`1`, `?`, `!`) vem primeiro e **nenhum registro abre com espaço**. Achado rodando o binário contra um repositório de verdade, não em teste — o fake devolvia o que se esperava dele.

### 6.2 Cobertura

**100% de statements em `internal/diff` e em `cmd/diff.go`.** Cada falha intermediária tem teste isolado — `Changes`, base, `read-tree`, `add`, `diff`, escrita do patch e escrita do resumo —, porque cada uma tem mensagem e caminho de saída próprios.

### 6.3 O cenário que define a feature

Árvore com três alterações, uma de cada natureza:

```text
cmd/branches.go    modificado      (tracked)
notas.txt          arquivo novo    (untracked)
logo.png           modificado      (tracked, binário)
```

| # | Cenário | Esperado |
|---|---|---|
| 1 | Export, aplicar em clone limpo | Os **três** reaparecem idênticos. É o teste que separa a feature de `git diff > x.patch`, que entrega 1 de 3 |
| 2 | `git status` antes e depois do export | Idêntico — `RN-01` |
| 3 | Conteúdo do índice real antes e depois | Byte a byte igual — `RN-01`, o risco concreto do `add -N` |
| 4 | Repo com `node_modules` ignorado | Fora do patch sem a flag; dentro com ela — `RN-03` |
| 5 | Patch corrompido artificialmente antes da verificação | Saída 1, **nada no `stdout`** — `RF-05`, `RN-02` |
| 6 | Árvore limpa | Mensagem específica, saída 0, patch vazio não emitido — `RF-01` |
| 7 | Fora de repositório git | Erro claro, saída 1 — `RF-06` |
| 8 | `apply` do `stdin` e de um arquivo | O patch inteiro chega ao `git apply` — `RF-07` |

A suíte cobre cada falha intermediária isoladamente — `Changes`, base, `read-tree`, `add`, `diff`, escrita do patch e escrita do resumo —, porque cada uma tem uma mensagem própria e um caminho de saída próprio.

### 6.4 Ponto de atenção do método

O cenário 1 é do tipo que a [SRS — equivalência](equivalencia.md) registra como armadilha: **teste de cenário precisa provar que o cenário existiu**. Se a montagem falhar em silêncio e a árvore ficar sem o binário, a asserção "os três voltaram" passa de graça. A pré-condição de cada um dos três arquivos precisa ser conferida antes.

---

## 7. Rastreabilidade

| Commit | Entrega |
|---|---|
| `6a4cbe8` | O port ganha `RunWithInputAndEnv` — o `GIT_INDEX_FILE` sem shell |
| `43c7c98` | `internal/diff`: `Changes`, `Export` e `Verify` sobre índice temporário — `RF-01`, `RF-02`, `RF-04`, `RF-05`, `RN-01` a `RN-03` |
| `3d7a876` | `gtr diff export` no `cmd` — `RF-01`, `RF-06` |
| `721f743` | README do `export` |
| `fc6f118` | `gtr diff apply`, domínio e cmd — `RF-07` |
| `b5c5a92` | README do `apply` |

---

## 8. Follow-ups conhecidos

- **O contrato `Selector` não generaliza, e a `RF-03` espera por ele.** A assinatura esboçada na **ADR-002** é `Select(candidates []branch.Branch) ([]branch.Branch, error)`. Arquivo não é branch. Ou o contrato vira genérico — `Selector[T any]` — ou nascem dois seletores quase iguais, que é a "terceira cópia" que a própria ADR aponta como onde a divergência começa. **A ADR-002 estava certa em adiar o contrato:** a segunda implementação o torceu antes de uma linha ser escrita.
- **O nome do comando não está fechado.** `gtr diff export` segue o padrão de subcomando do `gtr ignore list`, mas `diff` é substantivo emprestado do git e pode confundir com "mostrar o diff". Dado o histórico do projeto com nomes — ver a seção do `--yolo` na [SRS — equivalência](equivalencia.md) — vale decidir antes de mexer.
- **Exportar de uma entrada de stash** (`--stash <n>`), quando a feature de stash existir.
- **Formato de saída alternativo** — bundle em vez de patch, para carregar histórico junto. Não avaliado.
