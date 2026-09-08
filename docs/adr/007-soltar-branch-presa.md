---
titulo: ADR-007 — Soltar branch presa em worktree
data: 2026-08-07
status: entregue
escopo: gtr worktrees release, gtr worktrees remove
supersede: nada
---

# ADR-007 — Soltar branch presa em worktree

- **Data:** 2026-08-07
- **Status:** **entregue** — `remove` (commits `e8cc0a0`, `7b0ccf1`, docs `4e68156`) e `release` (commits `a73d0f0`, `c5c22b8`, docs `16e0f0a`). A proteção que só diagnostica veio antes, no `gtr branches`
- **Escopo:** `gtr worktrees release`, `gtr worktrees remove`
- **Supersede:** nada

---

## Contexto

O [`branches`](../srs/branches.md) protege branch em uso por outro working tree e **ensina** as três formas de soltar, sem executar nenhuma. Esta decisão trata de se, e como, a ferramenta pode executar.

### As três formas, verificadas empiricamente

| Caminho | O que perde | Recuperável? |
|---|---|---|
| `git -C <caminho> checkout --detach` | Nada — só move o `HEAD` | N/A |
| `git worktree remove <caminho>` | O diretório, **incluindo arquivos ignorados** | **Não** |
| `git worktree prune` | Só registro de diretório que já sumiu | N/A |

**Verificado que o `detach` preserva trabalho não commitado:** com um `.txt` novo e um arquivo modificado no working tree, os dois sobreviveram intactos ao `checkout --detach` executado de fora com `-C`.

**Verificado que o git se defende sozinho no caso ruim:** com rebase pela metade, o `detach` recusa com `error: you need to resolve your current index first`. Não é preciso escrever guarda — basta propagar o erro do git.

### O achado que muda tudo: a rede do `worktree remove` é mais frouxa do que parece

A intuição era que `git worktree remove` sem `--force` seria tão seguro quanto `git branch -d` sem `-D`, e que a ferramenta poderia confiar nele do mesmo jeito. **É falso**, e o teste é curto:

```console
worktree "limpo" para o git?  []   ← vazio, sim
$ git worktree remove ../linked
$ echo $?
0
.env sobreviveu?  NÃO — APAGADO
```

O `remove` recusa por arquivo **modificado ou não rastreado**. Arquivo **ignorado não é nenhum dos dois**, então o `.env` com credencial e o `node_modules/` inteiro foram apagados em silêncio, com exit 0.

| Comando | Recusa quando perderia algo irrecuperável? |
|---|---|
| `git branch -d` | **Sim** — e mesmo apagando, os commits ficam no reflog por cerca de 90 dias |
| `git worktree remove` | **Não** — ignorados não entram no critério |

A ferramenta confia no `-d` como segunda rede porque **aquela** rede recusa o irrecuperável. Estender a mesma confiança ao `remove` seria confiar numa garantia que ele não dá.

## Decisão

Duas features separadas, cada uma com o seu limite.

### `gtr worktrees release <branch>`

Executa **só o `detach`**, o caminho comprovadamente não destrutivo.

- Descobre o caminho do working tree a partir do nome da branch — o `List()` já devolve isso.
- Mostra **o que vai fazer e onde** antes de fazer.
- Pede confirmação com o padrão negativo do projeto.
- Avisa quando houver trabalho não commitado lá: ele sobrevive, mas passa a viver num `HEAD` destacado, e quem não conhece o estado se assusta.

**Três limites que fazem parte da decisão:**

1. **Nunca automático dentro do `branches --clean`.** Rodar um comando no repositório A e alterar o estado do diretório B é surpresa grande demais para acontecer como efeito colateral de limpar branch — a pessoa pode estar com aquele diretório aberto no editor agora. *Nunca destruir na dúvida* protege trabalho; esta protege **contexto**.
2. **O `branches` continua só diagnosticando.** Ele lista à parte e sugere; quem executa é um comando que a pessoa chamou para isso.
3. **Sem `--force`.** Se o git recusar, propaga o erro.

### `gtr worktrees remove <caminho>`

**Só faz sentido com guarda mais forte que a do git.** Embrulhar o `remove` sem acrescentar nada seria dar ao usuário a mesma armadilha com outro nome.

A guarda que faltava é exatamente a maquinaria da **[ADR-005](005-gtr-ignore.md)**:

```bash
git ls-files --others --ignored --exclude-standard
```

Com ela, o comando diz o que o git não diz:

```text
~/projeto-fix tem 3 arquivos ignorados que serão perdidos:
  .env
  node_modules/
  dist/
```

**Por isso a ordem entre as duas features era obrigatória:** antes da listagem de ignorados, o `gtr` só podia ensinar o comando.

O `IgnoredFiles` roda `git -C <caminho> ls-files --others --ignored --exclude-standard --directory --no-empty-directory -z` contra o working tree alvo, independente do diretório de quem chama; o preview mostra caminho e contagem antes da confirmação. O `Remove` é só `git worktree remove <caminho>`, **sem `--force`** — o que o git recusar continua recusado, propagado como erro.

## Alternativas consideradas

**Executar as três formas automaticamente no `--clean`.** Rejeitada com folga: efeito colateral em diretório que a pessoa não mencionou.

**Escolher uma forma por ela** — sempre `detach`, por exemplo. Rejeitada: as três custam coisas diferentes, e a escolha revela intenção. Quem quer o working tree de volta não quer o mesmo que quem terminou com ele.

**Embrulhar `worktree remove` confiando na guarda do git.** Rejeitada pelo achado acima.

**Só diagnosticar, nunca executar.** Defensável, e foi o estado por um bom tempo. O valor incremental do `release` é **modesto**: `git -C <caminho> checkout --detach` é um comando só, e o que o `gtr` acrescenta é não precisar descobrir o caminho nem sair do diretório. Real, mas pequeno — e é por isso que esta feature nunca foi prioritária.

## Consequências

**Positivas**

- Tira o trabalho manual do caso comum, que é justamente o não destrutivo.
- O `remove` do `gtr` fica **mais seguro que o git nativo**, que é o único argumento que justifica embrulhá-lo.

**Negativas**

- Superfície nova num comando que antes não tinha flag nenhuma.
- **Escopo em fuga:** isso começou como "filtrar uma branch da lista" e vira gerência de working tree.
- Alterar estado de outro diretório é categoria de ação que a ferramenta não tinha.

**Neutras**

- A proteção que só diagnostica já resolvia o problema real; estas duas são conveniência.

## Relacionadas

- **[SRS — worktrees](../srs/worktrees.md)** — o `List()` já devolve o caminho, que é metade do trabalho.
- **[SRS — branches](../srs/branches.md)** — a proteção que lista a branch presa à parte e ensina as três formas.
- **[ADR-005](005-gtr-ignore.md)** — pré-requisito do `remove`: sem a listagem de ignorados não havia guarda que justificasse.
