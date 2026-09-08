---
titulo: SRS — undo
data: 2026-08-15
status: entregue
comando: gtr undo
apelido: rewind
pacotes:
  - cmd/undo.go
  - internal/undo
  - internal/branch
  - internal/ui
commits:
  - 3697d34
fonte_externa: nenhuma
---

# SRS — undo

- **Data:** 2026-08-15
- **Feature:** `cmd/undo` + `internal/undo` + `branch.Restore`
- **Status:** **entregue** — commit `3697d34` na `main`
- **Fonte externa:** nenhuma. Só o `git` local e um arquivo dentro do diretório do git

---

## 1. Introdução

### 1.1 Propósito

Especificar o `gtr undo`, que **recria as branches que o `gtr branches --clean` deletou por último**.

A ferramenta apaga branch desde a primeira feature. A fronteira do que pode ser apagado é a lista, não a flag — `RN-01` da [SRS — branches](branches.md) —, e a cláusula 1 da **ADR-008** exige **prova de recuperação** antes de qualquer coisa destrutiva ir para a mão do usuário. Essa prova precisa ser implementação, não promessa escrita. É o que este documento especifica.

### 1.2 O git não oferece onde se apoiar, e isso foi medido

A suposição confortável seria "o reflog resolve". Não resolve:

```console
$ git branch -d feat
Deleted branch feat (was f6de15f).

$ ls .git/logs/refs/heads/
master            ← o reflog da feat foi apagado junto com ela

$ git reflog | head -2
f6de15f HEAD@{0}: merge feat: Fast-forward
6ed26b7 HEAD@{1}: checkout: moving from feat to master
```

Três achados:

- **O reflog da branch morre com a branch.** `.git/logs/refs/heads/<nome>` é removido na deleção.
- **O reflog do `HEAD` só tem a ponta se você esteve na branch.** Quem deletou sem nunca ter feito checkout nela não tem registro nenhum.
- **A única pista confiável é a frase que o git imprime** — `Deleted branch feat (was f6de15f)` — e ela é **prosa**, que a leitura de formato contratual proíbe como fonte — `RN-05` da [SRS — branches](branches.md).

Daí o diário próprio. E daí a ponta vir de `rev-parse --verify` **antes** de deletar, em vez de ser extraída daquela frase.

### 1.3 Escopo

**Entregue:** registro do que o `--clean` deletou; recriação do último lote com confirmação; as duas recusas; apelido `rewind`.

**Fora de escopo:** desfazer qualquer outra coisa. O `undo` não desfaz commit, merge, rebase ou stash — o git já tem `reset`, `revert` e `reflog` para isso, e embrulhá-los não acrescentaria guarda nenhuma.

### 1.4 O nome

**`undo`**, porque é a palavra que todo mundo já digita sem pensar.

**`rewind` é apelido**, e um easter egg: rebobinar a fita para antes da limpeza. Faz par com o `soundcheck` do `doctor` — os dois são termos de estúdio que **descrevem literalmente** o que o comando faz. Fica fora da ajuda da raiz.

---

## 2. Descrição geral

### 2.1 Posição na arquitetura

```text
internal/undo/               o diário — não conhece o que é branch
├── entry.go                 Entry: instante, ponta e nome
└── journal.go               Journal: Record e Last

internal/branch/
├── restoration.go           Restoration, ErrAlreadyExists, ErrGone
├── restore_result.go        RestoreResult
└── repo.go                  Restore, e o SHA capturado no Delete

internal/ui/restore.go       DescribeRefusal: a recusa em texto de tela
cmd/undo.go                  o comando, o alias e o relatório
```

**O `internal/undo` não sabe o que é branch.** Ele guarda instante, ponta e nome — três valores. Quem sabe recriar uma branch é o `internal/branch`; quem cruza os dois é o `cmd`, exatamente como já cruza `branch` e `worktree` para as branches presas.

### 2.2 O ciclo

| Momento | O que acontece |
|---|---|
| `--clean` confirma | `rev-parse` resolve a ponta de cada candidata **antes** de deletar |
| deleção | `branch -d` ou `-D`, como já era |
| logo depois | o que **de fato saiu** é gravado no diário, todas as linhas com o mesmo instante |
| `gtr undo` | lê o lote mais recente, mostra nome e ponta, pergunta, recria |

---

## 3. Requisitos funcionais

### RF-01 — Registrar o que foi deletado

Após uma deleção bem-sucedida, o `gtr` grava `<instante>\t<ponta>\t<nome>` em `<git-common-dir>/gtr/deleted`, em modo append. O instante é RFC 3339 em UTC.

**Todas as linhas de uma execução compartilham o instante**, e é isso — não um contador — que as torna um lote.

### RF-02 — Recriar o último lote

`gtr undo` lê o lote mais recente, **mostra o que vai voltar antes de perguntar**, e recria sob confirmação:

```text
Deletadas em 2026-08-15 15:56 (2):
  feat-a  0504044
  feat-b  6756275

Recriar 2 branches? [y/N]
```

A ponta abreviada está ali porque **é a única informação que permite reconhecer o trabalho** que está prestes a voltar. Nome sozinho não diz se é a versão que interessa.

Sem nada a desfazer, a saída é 0 e a mensagem é específica: *"Nada para desfazer: o gtr não deletou branch nenhuma neste repositório."*

### RF-03 — Recusar sem sobrescrever e sem prometer o impossível

| Situação | Resultado |
|---|---|
| já existe branch com aquele nome | **recusa** — `ErrAlreadyExists`, e a ref não é tocada |
| o commit não está mais no repositório | **recusa** — `ErrGone` |
| o git falha por outro motivo | erro propagado com o texto do próprio git |

Uma recusa **não aborta as seguintes**, e leva o comando a sair com 1. As duas recusas são distinguidas por `errors.Is`, não por texto.

### RF-04 — `rewind`

Apelido do `undo`, fora da ajuda da raiz e anunciado em `gtr undo --help`.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **O diário mora no `--git-common-dir`, nunca no `--git-dir`.** De dentro de um worktree o segundo aponta para `.git/worktrees/<nome>`, e **branch deletada é do repositório inteiro**. Conferido apagando no checkout principal e recuperando de dentro de um worktree. |
| **RN-02** | **Só entra no diário o que de fato saiu.** Branch cuja deleção falhou não é registrada. **Um `undo` que recria o que nunca foi apagado seria pior do que não ter `undo`**: destruiria a confiança na única rede de segurança da ferramenta. |
| **RN-03** | **Falhar ao registrar não desfaz a deleção.** Ela já aconteceu; fingir o contrário seria mentir. O comando avisa no `stderr` que aquelas branches ficaram **fora da rede**, e segue com saída 0. |

### 4.1 Regras herdadas

Estas não nascem aqui, e por isso vêm nomeadas e com o documento onde estão estabelecidas. **A numeração de `RF` e `RN` é local a cada documento** — um número sozinho não atravessa arquivos.

- **Nunca destruir na dúvida** — `RN-01` da [SRS — branches](branches.md), generalizada pela cláusula 1 da **ADR-008**. O `undo` é a contrapartida dela: agora existe prova de recuperação.
- **A leitura vem de formato contratual, nunca de saída para humano** — `RN-05` da [SRS — branches](branches.md). Aqui, a ponta vem de `rev-parse --verify --quiet`, nunca da frase `Deleted branch feat (was ...)`.
- **Nada no domínio escreve na tela** — `RN-11` da [SRS — branches](branches.md). `ErrAlreadyExists` e `ErrGone` são erros; o texto mora em `ui.DescribeRefusal`.
- **A recusa é acionável** — convenção do projeto: cada uma diz o que houve e o que fazer, em vez de só constatar a falha.

---

## 5. Achados de implementação

### 5.1 As três medições que fixaram o formato

| Pergunta | Medida | Consequência |
|---|---|---|
| Onde guardar? | Num worktree, `--git-dir` = `.git/worktrees/wt`; `--git-common-dir` = `.git` | `RN-01` |
| Que separador? | `check-ref-format` recusa espaço **e** tab em nome de branch | tab é seguro |
| O objeto sobrevive? | Sim, até `reflog expire` • `gc --prune` | `ErrGone` precisa existir |

### 5.2 A rede de segurança é mais fina justamente onde é mais necessária

O caso "commit podado" **não se reproduz** com o que o `--clean` puro apaga. A razão é óbvia depois de vista:

- **Branch mergeada por ancestralidade** — a que o `--clean` puro apaga — tem os commits **alcançáveis a partir da base**. O `gc` nunca os leva. Recuperação praticamente eterna.
- **Branch squashada ou rebaseada** — a que só o `--force` apaga — tem a ponta **inalcançável**. É essa que o `gc` poda, e é só com ela que o `ErrGone` acontece:

```console
$ git merge-base --is-ancestor <ponta-da-squashada> main
(falso)

$ gtr undo
  - squashada: o commit não está mais no repositório; o git já podou o que ficou inalcançável
```

**A assimetria:** o `undo` é infalível onde a deleção era segura, e falível onde a deleção era arriscada. Não é argumento contra o diário — é argumento **a favor** dele, e reforça o item em aberto sobre a premissa do `--force`.

### 5.3 Um `Close` que não pode ser engolido

O `Record` abre o arquivo, escreve e fecha — três caminhos de erro, dos quais dois são inalcançáveis em teste. Em vez de abrir um buraco de cobertura, a escrita e o fechamento saem juntos:

```go
_, err = handle.WriteString(lines.String())

return errors.Join(err, handle.Close())
```

**O `Close` de um arquivo aberto para escrita pode falhar sozinho**, e engoli-lo daria um diário truncado que se diz completo. O `errors.Join` mantém os dois e ainda deixa o statement alcançável.

### 5.4 Os testes de deleção contam deleções, não chamadas

A captura da ponta acrescenta um `rev-parse` por branch antes do `branch -d`, o que muda a **sequência** de chamadas ao git dentro do `Delete`. Teste que afirma a sequência exata quebra a cada guarda nova que se acrescente ali, sem que nada tenha regredido — a expectativa é que fica obsoleta, não o código.

Por isso `TestDelete` e `TestDeleteForcesOnlyEquivalentBranches` filtram da lista de chamadas só as que começam com `branch -`, e afirmam **essas** — quantas foram, em quais branches e com qual flag. O que mais o `Delete` precise perguntar ao git no meio do caminho não entra na expectativa.

### 5.5 Linha quebrada não pode derrubar o diário inteiro

O `parse` descarta em silêncio a linha truncada ou com instante ilegível, e segue para as outras. Um diário meio escrito — por disco cheio, por processo morto no meio da escrita — **não pode impedir de recuperar o que está bem escrito nas outras linhas**. É a mesma razão pela qual falhar ao registrar não aborta a deleção: o diário é rede de segurança, e rede de segurança que falha fechada não serve.

---

## 6. Testes

**100% de statements** em `internal/undo`, `internal/branch` e em `cmd/undo.go`. O `ui.DescribeRefusal` também está coberto — o pacote `internal/ui` inteiro não está, mas o que falta lá é de outro comando.

| # | Cenário | Esperado |
|---|---|---|
| 1 | Gravar e ler de volta | Mesma ordem da deleção — `RF-01`, `RF-02` |
| 2 | Duas limpezas gravadas | Só o lote mais novo volta — `RF-02` |
| 3 | Diário inexistente, vazio, e com linha quebrada | Sem erro; a linha boa sobrevive à ruim |
| 4 | Registrar nada | Não cria arquivo nenhum |
| 5 | Nome ocupado | Recusa, **e a ref não é tocada** — `RF-03` |
| 6 | Commit podado | Recusa distinta da anterior — `RF-03` |
| 7 | Uma recusa no meio de duas | A outra volta mesmo assim; saída 1 |
| 8 | Confirmação negada | Nenhuma ref criada, saída 0 |
| 9 | Entrada ilegível | Não pode ser lida como um sim |
| 10 | Deleção que falhou | Não entra no diário — `RN-02` |
| 11 | Diário não gravável | Deleção relatada, aviso no stderr, saída 0 — `RN-03` |
| 12 | `rewind` e a ajuda da raiz | Mesma saída; apelido não anunciado — `RF-04` |

**Conferido por mutação:** removendo a guarda de não sobrescrever, três testes caem; registrando também o que falhou ao deletar, um teste cai.

**E o ciclo inteiro foi rodado contra o git de verdade**, inclusive apagando no checkout principal e recuperando de dentro de um worktree.

---

## 7. Rastreabilidade

| Commit | Entrega |
|---|---|
| `3697d34` | O diário, o `Restore` com as duas recusas, o `--clean` gravando, o comando e o `rewind` — `RF-01` a `RF-04`, `RN-01` a `RN-03` |

Verificado pelo `gtr commits check` antes de ir para a `main`.

---

## 8. Follow-ups conhecidos

- **Só o último lote volta.** Desfazer a limpeza anterior à última exige editar o diário na mão. Um `--list` mostrando os lotes, e um seletor, resolveriam — e o seletor é o `Selector` do roadmap 4.
- **O diário cresce para sempre.** Uma linha por branch deletada, sem poda. Em anos de uso é ruído, não peso; medir antes de otimizar.
- **A recuperação não avisa que vai expirar.** O `doctor` poderia dizer "há N branches no diário cujo commit já sumiu", transformando o `ErrGone` em algo descoberto **antes** de ser preciso.
- **`--clean` com `--force` merece aviso mais forte.** É justamente o caso em que a rede é mais fina — §5.2.
