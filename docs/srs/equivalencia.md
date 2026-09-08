---
titulo: SRS — equivalência por conteúdo (squash e rebase)
data: 2026-08-05
status: entregue
comando: gtr branches --force
pacotes:
  - internal/branch
commits:
  - a766cf9
  - 346ae53
  - 53b0c4d
fonte_externa: nenhuma
---

# SRS — equivalência por conteúdo (squash e rebase)

- **Data:** 2026-08-05
- **Feature:** `internal/branch` — detecção de equivalência • flag `--force`
- **Status:** **entregue** — na `main`
- **Fonte externa:** nenhuma. **Este é o ponto:** o caso motivava a primeira feature de API, e foi resolvido só com o `git` local

---

## 1. Introdução

### 1.1 Propósito

Corrigir um **defeito de corretude** do `branches`: quando um PR é mergeado por squash ou por rebase, a branch nunca aparece na listagem, mesmo com todo o trabalho já integrado.

Não é aspereza como a de worktree — é a função principal errando por baixo, e a branch fica órfã para sempre porque nenhuma execução futura vai pegá-la.

**Agravante:** o CONTRIBUTING do próprio projeto prescreve exatamente squash ou rebase. No fluxo que o projeto recomenda, o comando lista menos do que deveria.

### 1.2 Escopo

- Detecção de branch integrada por **squash**.
- Detecção de branch integrada por **rebase**.
- Rótulo na listagem dizendo por qual caminho cada branch foi encontrada.
- Flag `--force` autorizando a deleção dessas branches.

**Fora de escopo:** consulta à API do GitHub (**ADR-006**), que era o contorno esperado e ficou desnecessário.

### 1.3 Definições

| Termo | Significado |
|---|---|
| **Patch-id** | Hash do diff normalizado de um commit — indiferente a SHA, autor, data e mensagem. Dois commits com o mesmo patch-id fazem a mesma mudança. |
| **Equivalente** | Branch cujo trabalho já está na base **por conteúdo**, ainda que não por ancestralidade. |
| **Commit virtual** | Commit descartável criado com `git commit-tree` para representar o diff acumulado de uma branch num objeto só. |
| **Mutante equivalente** | Mutação de teste que sobrevive por ser semanticamente idêntica ao original — categoria conhecida, não buraco de teste. |

---

## 2. Descrição geral

### 2.1 O que o git enxerga e o que não enxerga

`git for-each-ref --merged <base>` responde uma pergunta de **grafo**: o commit de ponta da branch é alcançável a partir da base?

Squash e rebase reescrevem o trabalho em commits novos — mesma árvore, paternidade diferente. A resposta é não, mesmo com o conteúdo inteiro integrado.

**A pergunta certa não é sobre paternidade, é sobre conteúdo:** o patch desta branch já está na base? `git cherry` responde exatamente isso, comparando por patch-id e marcando cada commit com `-` (existe equivalente) ou `+` (não existe).

### 2.2 Dois testes, porque um só não cobre os dois fluxos

Verificado empiricamente em repositórios descartáveis, com pré-condição conferida antes de cada asserção:

| Cenário | `--merged` | `cherry` por commit | `cherry` do virtual combinado |
|---|---|---|---|
| Squashada, vários commits | não vê | `+` não pega | **`-` pega** |
| Rebaseada, vários commits | não vê | **`-` pega** | `+` não pega |
| Trabalho de verdade | não vê | `+` | `+` |
| Branch vazia | vê (já é ancestral) | zero linhas | `+` |

**Os dois testes são complementares, não redundantes** — implementar só um deixaria metade do defeito de pé:

- **Squash** junta N commits em um. Nenhum commit individual da branch casa, mas o **diff acumulado** casa.
- **Rebase** replica os N commits um a um. Cada um casa, mas nenhum commit na base carrega o diff combinado.

### 2.3 O commit virtual

```bash
mergeBase=$(git merge-base <base> <branch>)
virtual=$(git commit-tree $(git rev-parse <branch>^{tree}) -p $mergeBase -m "gtr equivalence probe")
git cherry <base> $virtual        # "- <sha>" = squashada
```

**Por que `commit-tree` e não `git patch-id`.** O caminho aparentemente mais direto seria mandar o diff para o `patch-id` e comparar à mão. Mas o `patch-id` lê do **stdin**, e o contrato do `Runner` é `Run(args ...string)`, sem como alimentá-lo. `commit-tree` + `cherry` resolve **sem tocar no contrato** que a suíte inteira usa. O port ganhou `RunWithInput` depois (commit `2504eaf`, pela dívida do `gtr ignore list` — **ADR-005**), e ainda assim o contorno **segue valendo**: ele nunca foi dívida técnica, e sim a decisão de não mexer no contrato de toda a suíte por causa de uma feature só.

**Custo assumido: esta é a primeira vez que o `gtr` escreve no banco de objetos.** O `commit-tree` grava um objeto solto, sem referência, que o `git gc` recolhe. Não é escrita no working tree, não é escrita em ref, não é destrutivo — mas é escrita, e a promessa de "só executa git e imprime" fica com essa ressalva.

**Verificado sobre o `commit-tree`:** não respeita `commit.gpgsign` (testado com assinatura forçada e chave inexistente — mesmo SHA, sem erro), então não há risco de travar num prompt de senha; e **falha** se não houver identidade configurada, o que degrada para "não equivalente" em vez de derrubar o comando.

---

## 3. Requisitos funcionais

### RF-01 — Enxergar branch integrada por squash ou rebase

`gtr branches` lista também as branches cujo trabalho já está na base **por conteúdo**, e informa por qual caminho cada uma foi encontrada: `mergeada`, `squashada` ou `rebaseada`.

A detecção é **conservadora por construção**: o que não for provado equivalente fica de fora.

### RF-02 — Deletar as equivalentes sob `--force`

`gtr branches --clean` sozinho **não apaga** as squashadas e rebaseadas: o `git branch -d` as recusa, e o comando informa quantas ficaram de fora em vez de contornar a recusa em silêncio.

Com `--clean --force` elas entram, e só aí o `-D` é usado — exclusivamente nelas. Branch mergeada por ancestralidade continua saindo por `-d` mesmo com a flag ligada.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **Verdade vácua é barrada.** "Todos os commits têm equivalente" é vacuamente verdade quando não há commit nenhum. `git cherry` devolve zero linhas nesse caso, então o teste de rebase exige **pelo menos uma linha** além de todas serem `-`. Mutação confirmou que a guarda é necessária. |
| **RN-02** | **Resposta inesperada não conta.** O commit virtual está sempre sozinho à frente do merge-base, então o `cherry` deve devolver exatamente uma linha; mais que isso é sinal de que a premissa quebrou, e o resultado é descartado. |
| **RN-03** | **Falha degrada, não aborta.** `merge-base`, `rev-parse` e `commit-tree` falhando devolvem "não equivalente". Uma branch problemática não derruba a listagem inteira. **Errar para menos é seguro** (deixa de oferecer); errar para mais apaga trabalho. |
| **RN-04** | **Protegida não é sondada.** A base, `main`, `master` e a atual saem antes da sondagem — economiza `git` e mantém as duas proteções da [SRS — branches](branches.md) (`RN-02` e `RN-03` de lá) valendo por construção. |

---

## 5. Por que o `--force` teve de existir

Detectar não bastava. **O `git branch -d` recusa branch que não seja ancestral da base** — inclusive as que a detecção acabou de provar integradas:

```console
$ git branch -d feat
error: the branch 'feat' is not fully merged.
If you are sure you want to delete it, run 'git branch -D feat'
```

Uma proibição categórica do `-D` tornaria a feature impossível de completar: o comando listaria branches que ele mesmo não conseguiria apagar.

**A garantia que importa não é "não usar `-D`", e sim "não destruir trabalho não integrado"** — e essa fica inteira, por um motivo estrutural: o `branches` nunca lista branch com trabalho pendente, então **a lista é a fronteira de segurança, não a flag**. O `-D` só alcança o que a comparação de conteúdo já provou redundante. É a `RN-01` da [SRS — branches](branches.md) na forma em que ela vale hoje.

### 5.1 O nome da flag — `--yolo` proposto e descartado

A flag nasceu `--yolo`, e o nome não sobreviveu. O motivo não é que piada em CLI seja proibida — o nome do projeto inteiro é um trocadilho. O motivo é que **o nome descrevia errado o risco**.

"You only live once" promete operação temerária e sem rede. A flag faz o oposto: só alcança branch **provada** redundante. Alarme falso ensina que a ferramenta está fazendo algo duvidoso quando ela está fazendo algo rigoroso. Os precedentes reais desse estilo — `pip --break-system-packages`, `--dangerously-skip-permissions` — nomeiam perigos **ilimitados e não verificados**; este é limitado e verificado.

Três consequências pesaram: o `--help` não ensinava nada; a flag vira registro em runbook e CI, e a regra do projeto é **não apelidar o que vira registro**; e gastava o nome alarmante no caso seguro, deixando o próximo perigo de verdade sem nome disponível.

---

## 6. Testes

**100% mantido em `internal/branch`.** Seis mutações deliberadas sobre a detecção:

| Mutação | Resultado |
|---|---|
| Guarda de `cherry` vazio removida (verdade vácua) | Pegou |
| `-D` usado sempre que `--force` está ligado | Pegou |
| `squashada` e `rebaseada` trocadas | Pegou |
| Protegidas entram na sondagem | Pegou |
| `cherry` trata `+` como presente | Pegou |
| `len(present) == 1` vira `>= 1` no teste do squash | **Sobreviveu** — buraco de teste, fechado com subteste próprio |

**A cobertura pegou um teste mentindo sobre o próprio cenário.** Um subteste chamado "commit-tree falhando" roteirizava, na verdade, o `cherry` falhando — o `commit-tree` continuava respondendo com sucesso. Descoberto porque a cobertura ficou em 96,6% com exatamente aquelas linhas de erro sem executar. Mesma família da branch `feat;touch OWNED`: cenário que não existiu, asserção que passou de graça.

### 6.1 Um mutante equivalente, registrado

Trocar `repo.Delete(candidates, options.force)` por `repo.Delete(candidates, true)` é indistinguível de fora:

- com `--force`, a lista contém equivalentes e as duas versões forçam;
- sem `--force`, o `deletable` só devolve mergeadas por ancestralidade, e o `Delete` nunca usa `-D` nelas.

Passar `options.force` continua certo — é a defesa se aquele invariante regredir —, mas nenhum teste externo pode prová-lo. **Mutante equivalente não deve ser "fechado" com asserção artificial**; fica registrado para não ser reinvestigado a cada bateria.

---

## 7. Rastreabilidade

| Commit | Entrega |
|---|---|
| `a766cf9` | Nasce `MergeKind`; `Merged` varre todas as refs locais e sonda as que o `--merged` não trouxe — `RF-01`, `RN-01` a `RN-04` |
| `346ae53` | `Delete` ganha o parâmetro de força e escolhe `-d`/`-D` por branch — `RF-02` |
| `53b0c4d` | README: a seção "o que a ferramenta nunca faz" passa a afirmar a garantia estreita — não destruir trabalho não integrado — no lugar da ampla, que esta feature tornou falsa |

---

## 8. Contornos avaliados e descartados

Três contornos foram avaliados antes deste. **Nenhum foi o escolhido**, e dois estavam errados:

| Contorno | Veredito |
|---|---|
| (a) `upstream:track` = `gone` | **Erra para mais**, que é a direção inaceitável — remota apagada sem merge marcaria a branch como limpável |
| (b) "diff vazio contra o merge-base" | **Erra nas duas direções**, verificado: `git diff <base>...<branch>` marca a branch vazia e **não** marca a squashada |
| (c) API do GitHub | Exato, mas custa token, rede e a reescrita do requisito não-funcional de operação local. **Ficou desnecessário** |

**A branch vazia, que se previa como falso positivo do patch-id, não é** — também verificado.

**Lição de método:** antes de pagar token, rede e reescrita de requisito não-funcional, vale procurar se o git local já sabe a resposta. Neste caso sabia, e o `git cherry` existe desde sempre para exatamente isso.

---

## 9. Follow-ups conhecidos

- **Ambiguidade de um commit só** — numa branch com um único commit, squash e rebase produzem o mesmo resultado e o patch combinado *é* o patch do commit. O rótulo cai em `squashada` por desempate, escolhido porque "Squash and merge" em PR de um commit é mais comum. Cosmético — a decisão de listar está certa nos dois casos.
- **Custo não medido** — 4 a 5 invocações de `git` por branch ociosa. A opção de desligar não foi implementada por falta de medição. Medir antes de otimizar.
