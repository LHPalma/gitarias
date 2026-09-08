---
titulo: SRS — árvore de branches
data: 2026-08-13
status: entregue
comando: gtr branches --tree
pacotes:
  - cmd/branches.go
  - cmd/tree_table.go
  - internal/branch
  - internal/ui
commits:
  - d79fdb8
  - 4301e77
  - fcd44d0
  - 121dc4e
fonte_externa: nenhuma
---

# SRS — árvore de branches

- **Data:** 2026-08-13
- **Feature:** `cmd/branches --tree` + `internal/branch`
- **Status:** **entregue** — na `main`
- **Fonte externa:** nenhuma — só o `git` local

---

## 1. Introdução

### 1.1 Propósito

Especificar o `gtr branches --tree`, que mostra **todas** as branches locais como árvore, cada uma sob aquela em que foi empilhada.

**O git não registra quem é o pai de uma branch.** `branch.<nome>.merge` aponta para o **upstream remoto**, não para a branch local de onde o trabalho saiu. A relação existe no grafo, mas não tem nome nem lugar onde ser lida.

Daí a família de ferramentas de PR empilhado — e a feature nativa do GitHub, em preview público desde 2026-07-30 — **guardarem essa informação por fora**: `.git/gh-stack` é um JSON local com a ordem das camadas. É metadado paralelo para descrever algo que o próprio grafo já diz.

### 1.2 O que se mediu antes de decidir construir

A pergunta era se havia algo a construir em cima das stacked PRs do GitHub. A medição veio antes da resposta: uma pilha de três camadas foi montada e mergeada por squash, com rebase em cascata entre elas, reproduzindo o que o `gh stack` produz. O `git branch -d` recusou as três; o `gtr branches` detectou as três como `squashada`.

**A ferramenta já resolvia o rastro que uma stack deixa**, sem uma linha nova — e stacks multiplicam a frequência desse problema, porque uma mudança passa a deixar N branches órfãs em vez de uma.

O que faltava não era ler o arquivo do `gh stack`: era o **conceito** que a feature popularizou. Verificar que a estrutura é inferível do grafo levou três comandos de shell e reconstruiu a pilha inteira. **O jeito honesto de aproveitar uma ferramenta nova é pegar o conceito, não o arquivo.**

### 1.3 Escopo

- `gtr branches --tree` — todas as branches locais, como árvore, com o estado de cada uma.
- Inferência do pai imediato a partir do grafo.
- Saída estruturada com coluna `pai` — **ADR-004**, sem código novo de formatação.

**Fora de escopo, por decisão e não por sequenciamento:**

- **Ler o `.git/gh-stack`.** É estado privado de um produto em preview, sem esquema documentado, não commitado, e exigiria que o `gh stack` estivesse instalado. Ver §2.2.
- **Qualquer chamada à API do GitHub.** Saber o estado de um PR é o roadmap 7, **ADR-006**, deliberadamente por último.
- **Editar a pilha.** Reordenar, inserir ou dobrar camadas é o que o `gh stack modify` faz. Esta feature **lê**.

### 1.4 Definições

| Termo | Significado |
|---|---|
| **Camada** | Uma branch dentro da árvore, com o pai em que foi empilhada e o estado de merge. É o `branch.Layer`, distinto do `branch.Branch` que a listagem normal usa. |
| **Pai** | A branch local de onde outra saiu. **Não é o upstream** e não tem registro no git; é inferido. Pai vazio significa que a camada pendura direto na base. |
| **Pilha meio pousada** | Estado em que as camadas de baixo já foram mergeadas e as de cima seguem abertas. É o caso que a listagem plana não distingue de branches independentes. |
| **Rebase em cascata** | O que o `gh stack rebase` faz quando uma camada pousa: reescreve todas as de cima sobre a nova base. Efeito colateral relevante: depois dele, as camadas deixam de estar empilhadas umas nas outras e passam a pendurar direto na base. |

---

## 2. Descrição geral

### 2.1 A inferência

Se a **base de merge de B é exatamente a ponta de A**, então B foi começada a partir de A.

O cuidado está em qual A escolher: numa pilha de três, `camada-1` **também** é ancestral da `camada-3`, e a base também é. Pegar o primeiro candidato encontrado desenha a árvore errada — achata a pilha. O pai é o **mais próximo**.

Implementação: um mapa de ponta → branch, montado com um `for-each-ref`, e um `rev-list <branch> ^<base>` por branch. Caminhando do topo da branch para trás, **o primeiro commit que é ponta de outra branch é o pai imediato** — a ordem do `rev-list` já entrega isso de graça, sem contar distâncias.

```text
internal/branch/
├── layer.go                 Layer: branch, pai e se está mergeada
└── repo.go                  classify, Tree, tips, parentOf

internal/ui/branch.go        DescribeLayer: o rótulo da camada
cmd/tree_table.go            a árvore e a tabela plana
cmd/branches.go              a flag --tree
```

**Custo:** um `rev-list` por branch, mais um `for-each-ref`. Mesma ordem de grandeza da detecção de equivalência, que já gasta 4 a 5 invocações por branch não mergeada.

### 2.2 Por que não ler o `.git/gh-stack`

| | Ler o JSON | Inferir do grafo |
|---|---|---|
| Fidelidade | exata — é a fonte do `gh stack` | reconstrói o que o grafo mostra |
| Dependência | exige `gh stack` instalado **e usado** | nenhuma |
| Estabilidade | JSON privado, produto em preview, esquema não documentado | `merge-base` não muda desde sempre |
| Alcance | só quem usa a feature | quem empilha com `rebase --onto` também, e isso é anterior à feature |

**E um argumento que só aparece depois de implementar:** o JSON guarda a **ordem** das camadas e o contexto do repositório. A ordem é justamente o que o grafo já dá. Não haveria informação nova — só acoplamento.

---

## 3. Requisitos funcionais

### RF-01 — Desenhar a árvore das branches locais

`gtr branches --tree` imprime a base como raiz e, sob ela, todas as branches locais aninhadas segundo o pai inferido, com o estado de merge de cada uma alinhado à direita.

Sem branch nenhuma além da base: mensagem específica, saída 0.

**A base é a raiz, nunca uma camada.**

### RF-02 — Inferir o pai imediato

O pai de uma branch é a branch local cuja ponta é o ancestral **mais próximo** dela. Não havendo nenhuma, a branch pendura na base.

Uma branch nunca é pai de si mesma, e a base nunca aparece como pai nomeado — pendurar nela é o pai vazio.

**Falha do git ao ler o histórico de uma branch não derruba a árvore:** aquela camada pendura na base e o resto sai inteiro. Árvore parcial vale mais que erro.

### RF-03 — Mostrar todas, e não só as mergeadas

A listagem sem `--tree` mostra apenas o que está mergeado, porque responde *o que eu posso limpar*. A árvore responde *como minhas branches se relacionam* e mostra todas, cada uma com seu estado — `mergeada`, `squashada`, `rebaseada` ou `não mergeada`.

É a mesma lógica que separou a tabela da seção de branches presas na [SRS — branches](branches.md): pergunta diferente, forma diferente.

### RF-04 — Emitir a árvore em formato estruturado

`--format text|csv|tsv|json`, com `--output`, `--separator` e `--no-header` — **ADR-004**.

- **csv/tsv:** `branch`, `pai`, `estado`. A hierarquia vira coluna, ordenada da base para o topo, o que basta para um script remontar a árvore.
- **json:** envelope com a chave `layers`, `parent` sempre nomeado, `merged` booleano e `merge` como token — vazio quando a camada não está mergeada.

**Na tabela o pai vazio vira o nome da base.** Vazio não diz nada para quem lê uma planilha; na árvore a raiz já está desenhada e a informação é redundante.

### RF-05 — Recusar `--tree` com `--clean`

A árvore lista **de propósito** o que não pode ser deletado. Cruzar as duas flags só confundiria sobre o que a confirmação vai apagar. Mesma postura de `--format` com `--clean`.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **O pai vem do grafo, nunca do estado de outra ferramenta.** Nada de ler `.git/gh-stack`, `refs/stk/*` ou qualquer metadado paralelo. A relação está no grafo e o grafo é estável; o resto é formato privado de produto em preview. Consequência: funciona para quem nunca instalou ferramenta de PR empilhado. |
| **RN-02** | **O pai é o ancestral mais próximo, não um ancestral qualquer.** Numa pilha, toda camada de baixo é ancestral das de cima, e a base é ancestral de todas. Pegar o primeiro candidato acha a árvore errada — desenha uma pilha de três como três irmãs. |
| **RN-03** | **Duas perguntas, dois escopos.** A listagem mostra o mergeado, a árvore mostra tudo. Uma árvore que só mostrasse o mergeado perderia exatamente o caso que a justifica: a pilha meio pousada, em que as camadas de cima ainda estão abertas. |
| **RN-04** | **Inferência parcial vale mais que erro.** Branch cujo histórico o git não consegue ler pendura na base, e a árvore sai inteira. O objetivo é orientação, e uma árvore com um galho no lugar errado ainda orienta; nenhuma árvore não. |

### 4.1 Regras herdadas

- **Protegida não é sondada**, nem para a árvore — `RN-04` da [SRS — equivalência por conteúdo](equivalencia.md). A guarda existia para a listagem e sobreviveu à refatoração **porque tinha teste**: ver §6.1.
- **A leitura vem de formato contratual** — `RN-05` da [SRS — branches](branches.md). `for-each-ref` com `--format` e `rev-list`, ambos plumbing; `LANG` em qualquer idioma não quebra o parse.
- **Árvore no `stdout`, base no `stderr`** nos formatos delimitados — `RN-09` da [SRS — branches](branches.md), mesmo tratamento que a listagem já dá.
- **Nenhuma linha termina em espaço** — `RN-10` da [SRS — branches](branches.md), inclusive as com prefixo de árvore.
- **O domínio não desenha nada** — `RN-11` da [SRS — branches](branches.md). `internal/branch` devolve `[]Layer` com o pai como string; os conectores e o rótulo do estado vivem no `cmd` e no `internal/ui`.

---

## 5. Interface

```console
$ gtr branches --tree
Base: main (encontrada localmente)

main
└─ camada-1        squashada
   └─ camada-2     não mergeada
      └─ camada-3  não mergeada
```

```console
$ gtr branches --tree --format csv
branch,pai,estado
camada-1,main,squashada
camada-2,camada-1,não mergeada
camada-3,camada-2,não mergeada
```

| Flag | Padrão | Efeito |
|---|---|---|
| `--tree` | `false` | Todas as branches locais como árvore, em vez da lista das mergeadas |

---

## 6. Testes

**100% de statements em `cmd`, `internal/branch` e `internal/ui`**, com a suíte inteira rodando contra o fake.

### 6.1 A regressão que a suíte não pegou

A extração do `classify` — separar *como a branch chegou na base* de *quais nunca podem ser deletadas* — mudou a **ordem da listagem**. As mergeadas por ancestralidade vinham antes das equivalentes; o passe único passou a intercalar.

**A suíte ficou verde com a ordem errada.** Quem pegou foi a comparação com o binário anterior, que é o método exigido de todo commit `refactor:`. A guarda que faltava entrou junto com a correção, e foi conferida invertendo o laço e vendo o teste reprovar.

**A guarda que existia funcionou:** o teste de que branch protegida não é sondada reprovou na hora em que a filtragem saiu de lugar, e forçou a passar o conjunto de protegidas para o `classify`.

As duas coisas na mesma refatoração, e o contraste é o registro: **o que tem teste sobrevive a refatoração; o que não tem, sobrevive por sorte.**

### 6.2 Cenários

| # | Cenário | Esperado |
|---|---|---|
| 1 | Pilha de três mais uma branch solta | Cada camada sob a anterior, a solta sob a base — `RF-02` |
| 2 | A camada de cima tem duas ancestrais | O pai é a mais próxima, e o teste falha se for a outra — `RN-02` |
| 3 | Pilha meio pousada | A de baixo mergeada, as de cima abertas, todas na árvore — `RF-03` |
| 4 | Comparação entre a lista plana e a árvore | A árvore mostra o que a lista esconde. **Confirma antes que a lista de fato esconde**, senão passa de graça |
| 5 | `rev-list` falhando para uma branch | Aquela camada pendura na base, as outras saem certas — `RN-04` |
| 6 | Ponta repetida e ponta igual à da base | Nunca pai de si mesma, nunca a base nomeada — `RF-02` |
| 7 | Camada cujo pai não está na lista | Continua na tabela; a ordenação por caminhada cai para a lista bruta |
| 8 | Linha quebrada e linha em branco na saída do git | Ignoradas sem atrapalhar o resto |
| 9 | `--tree` com `--clean` | Erro antes de tocar no git — `RF-05` |

### 6.3 Conferido contra o git de verdade

Três repositórios montados à mão: pilha aberta, pilha inteiramente mergeada por squash com rebase em cascata, e pilha meio pousada. Os blocos do README saem de dois deles.

**Um achado do cenário da pilha inteiramente mergeada:** depois do rebase em cascata, as camadas **deixam de estar empilhadas** — cada uma passa a pendurar direto na base, porque foi rebaseada sobre ela. A árvore mostra três irmãs, e isso está **certo**: reflete o grafo depois da cascata, não a intenção original.

---

## 7. Rastreabilidade

| Commit | Entrega |
|---|---|
| `d79fdb8` | `classify` sai de dentro do `Merged`, preservando a ordem da listagem. Nasce a guarda que faltava — §6.1 |
| `4301e77` | Nasce `Layer` e a inferência do pai. `ui.DescribeLayer` — `RF-02`, `RN-01`, `RN-02`, `RN-04` |
| `fcd44d0` | `--tree`: a árvore, a tabela plana e os quatro formatos — `RF-01`, `RF-03`, `RF-04`, `RF-05` |
| `121dc4e` | README. Blocos rodados contra repositórios montados para eles |

A divisão em commits foi **reprovada pelo `gtr commits check`** na primeira tentativa: o `Tree()` ficou num commit e o tipo `Layer` no seguinte, e o primeiro não compilava sozinho.

---

## 8. Follow-ups conhecidos

- **A premissa do `--force` mudou, e vale reexaminar.** Hoje o `--clean` sozinho não apaga squashada nem rebaseada; exige `--force`. Isso foi desenhado quando squash era um caso entre outros. **Com stacks, squash vira o caso normal** — cada camada chega na base squashada —, e o `--force` deixa de ser válvula de exceção para virar o que se digita sempre, que é o começo do fim de qualquer guarda. Não é proposta de afrouxar; é registro de que a premissa se moveu.
- **Ordem de deleção numa pilha não é considerada.** O `--clean` deleta na ordem em que a listagem produziu. Numa pilha isso não perde dado — os commits seguem alcançáveis pelas camadas de cima —, mas apagar a de baixo primeiro remove o marcador de onde a pilha começa. Não avaliado se vale ordenar.
- **Custo não medido.** Um `rev-list` por branch soma à detecção de equivalência, que também não foi medida. Medir as duas juntas, ou nenhuma.
- **`--tree` mostra branch protegida como "não mergeada"** quando ela está squashada, porque a sondagem é pulada nela. Correto quanto ao custo, impreciso quanto ao rótulo. Um estado `protegida` na camada resolveria.
- **A árvore não distingue a branch atual.** O `worktrees` marca a dele com `*`; aqui não há marcação equivalente.
