---
titulo: "ADR-008 — Operações destrutivas: prova, autorização e alcance"
data: 2026-08-07
status: adotada
escopo: transversal — toda operação que destrói ou reescreve
supersede: a leitura de que as regras de branches proíbem operação destrutiva
---

# ADR-008 — Operações destrutivas: prova, autorização e alcance

- **Data:** 2026-08-07
- **Status:** **adotada** — toda operação destrutiva entregue desde então cumpre as três cláusulas, e a única exceção, o [`gtr fire`](../srs/fire.md), declara por que é exceção
- **Escopo:** transversal — `branches`, `worktrees`, `ignore`, `author`, `overdub`, `ai-trailers strip`, a feature de stash e qualquer escrita em ref remota
- **Supersede:** a leitura de que as regras do `branches` proíbem operação destrutiva

---

## Contexto

### O projeto já tomou esta decisão duas vezes sem escrevê-la

Duas regras nasceram como proibição absoluta, e as duas precisaram ser reescritas pelo mesmo motivo.

A primeira dizia que o `git branch -D` não estava exposto "nem atrás de flag". Colidiu com a detecção de branch squashada, e a reescrita deixou a lição no próprio corpo: *a garantia que importa nunca foi "não usar `-D`", e sim "não destruir trabalho não integrado"*.

A segunda dizia que a ferramenta "nunca toca em branch remota", e a especificação ecoava isso como *fora de escopo indefinidamente*. Colidiu com a intenção declarada de a rede existir um dia — o que sempre esteve no plano.

O erro é o mesmo nas duas: **escrever a mecânica proibida no lugar da garantia pretendida.** Proibição de mecanismo envelhece mal, porque o mecanismo reaparece legítimo e a regra vira obstáculo ao próprio roadmap. Garantia sobrevive à feature que a testa.

O `worktrees remove` já obedecia a esta decisão antes de ela existir: ele esperou o `ignore list` porque, sem enxergar os arquivos ignorados que seriam apagados junto, não havia guarda que justificasse embrulhar o comando do git.

### Por que flag e confirmação não bastam

A formulação tentadora é "tudo que é destrutivo mora atrás de flag obrigatória e confirmação". Ela é necessária e **insuficiente**.

Confirmação é a proteção mais fraca que existe: é o prompt que se aperta sem ler, ainda mais em ferramenta de uso diário. E flag é declaração de **intenção**, não de segurança — `--force` diz "eu quero", nunca "isto é recuperável".

A força real da regra do `branches` nunca esteve no `--force`. Está em **"a fronteira do que pode ser apagado é a lista, não a flag"**: o `--force` só alcança branch que a comparação de patch-id já provou redundante. Trocar essa cláusula por um prompt seria abrir mão da única parte que protege de verdade.

## Decisão

Toda operação destrutiva do `gtr` cumpre **as três cláusulas**, não uma delas.

| # | Cláusula | O que exige |
|---|---|---|
| 1 | **Prova de recuperação** | Antes de destruir, a ferramenta **demonstra** que o conteúdo continua alcançável — por ancestralidade, por patch-id, ou por uma cópia que ela mesma criou e releu. Demonstrar não é "o comando de salvar retornou 0". O que não passa na prova **não entra na lista**, e o que não está na lista não é destruído por flag nenhuma. |
| 2 | **Autorização explícita** | Flag obrigatória, nunca comportamento padrão, e confirmação com **padrão negativo**. Nada destrutivo acontece por omissão nem por inferência do que o usuário "provavelmente quis". |
| 3 | **Alcance dimensiona a guarda** | Quanto mais gente a operação atinge, mais específica a confirmação. Operação que afeta só quem rodou pode confirmar por contagem. Operação que afeta terceiros **nomeia cada alvo**, e usa **flag própria**, jamais herdada do caminho local — para que ninguém alcance a rede por hábito de digitar a flag de sempre. |

### A ordem faz parte da decisão

```text
provar  →  autorizar  →  destruir
```

As três nunca se fundem num passo só. É a mesma disciplina que a **[ADR-002](002-dois-front-ends.md)** fixou para `filtrar → Select → Delete`, e pelo mesmo motivo: quando prova e autorização se misturam, aparece a pergunta sem resposta boa — o que fazer quando o usuário autoriza algo que a prova não cobre. Mantendo a prova antes, essa pergunta não chega a existir.

### Como cada operação cumpre

| Operação | Prova | Autorização | Alcance |
|---|---|---|---|
| `branches --clean` | Ancestralidade: a ponta da branch é alcançável da base | `--clean` • `[y/N]` | Só quem rodou. Confirmação por contagem basta |
| `branches --clean --force` | Patch-id: conteúdo provado redundante | `--force`, além do `--clean` | Só quem rodou |
| [`author`](../srs/author.md) e [`overdub`](../srs/overdub.md) | O `HEAD` anterior impresso antes da pergunta, para o `reset --hard` | Confirmação obrigatória, sem `--force` | Só quem rodou; a prévia diz quantos commits e quais autores |
| [`ai-trailers strip`](../srs/ai-trailers-strip.md) | Mesma linha de recuperação, sobre o mesmo mecanismo | Confirmação obrigatória | Só quem rodou |
| Largar o diff da árvore (feature de stash) | Cópia em `refs/stash` criada **e relida** antes de limpar | Flag própria + confirmação | Só quem rodou. **Atenção aos untracked:** `git diff` não os captura, então limpá-los sem tê-los salvo viola a cláusula 1 |
| [`worktrees remove`](../srs/worktrees.md) | Listar os ignorados que seriam apagados | Flag própria + confirmação | Só quem rodou, mas atinge arquivo **não versionado**, que não tem recuperação nenhuma: a prova aqui é mostrar, não garantir |
| Deletar ref remota | Conteúdo contido na base, mesma comparação do caminho local | Flag **própria**, que não se confunde com `--force` | Todos que derem pull. **Cada ref nomeada** na confirmação; nunca lote por padrão |

### A exceção, e por que ela é uma só

O [`gtr fire`](../srs/fire.md) não pede confirmação. É deliberado, está no `--help` dele, e se sustenta porque a operação **não destrói**: ela salva — `add -A`, commit e push para uma ref nova, que nunca sobrescreve o que já estava no remoto. A cláusula 2 é dispensada porque um botão de pânico que pergunta não serve ao próprio propósito; as outras duas continuam de pé.

## Alternativas consideradas

**Manter a proibição por mecanismo** (status quo). Rejeitada por evidência: falhou duas vezes, e nas duas o custo foi uma regra que impedia trabalho legítimo até alguém reescrevê-la sob pressão de uma feature. Regra que só é revista quando atrapalha não está protegendo — está atrasando.

**Só flag e confirmação.** Adotada **em parte**: virou a cláusula 2. Rejeitada como guarda única porque deixaria a segurança dependendo de o usuário ler o prompt, e porque descartaria a cláusula que já se provou valer — a fronteira ser a lista.

**Exigir backup automático antes de toda destruição.** Rejeitada por ruído: para branch que a ancestralidade já provou contida, um backup seria cópia de algo que não se perde. A prova **é** a recuperação quando o conteúdo já está em outro lugar; a cópia só é necessária quando não está, que é exatamente o caso da feature de stash.

**Classificar operações em níveis de perigo** (baixo, médio, alto). Rejeitada por ser régua sem critério: "alto" vira opinião. As três cláusulas produzem a mesma graduação a partir de fatos verificáveis — existe prova? quem é atingido?

## Consequências

**Positivas**

- Destrava a escrita em ref remota, a feature de stash e o `worktrees remove`, sem abrir mão de garantia.
- Dá critério para features futuras não previstas: quem propuser algo destrutivo responde três perguntas objetivas em vez de negociar exceção.
- Torna as regras do `branches` instâncias de um princípio, em vez de regras avulsas que se contradizem.

**Negativas**

- A cláusula 1 é a mais cara de implementar e a mais fácil de fingir. "Salvei e deu certo" não é prova; reler custa código e teste.
- Operações sobre arquivo não versionado nunca cumprem a cláusula 1 de verdade — no máximo mostram o que será perdido. Isso está assumido, não resolvido.

**Neutras**

- Nada muda no comportamento de quem já cumpria: `branches --clean` e `--force` sempre cumpriram as três.

## A cláusula 1 era dívida, e foi paga

Por muito tempo a prova de recuperação foi promessa escrita: o `gtr` deletava branch desde a primeira feature e não havia como desfazer. **O git não dá apoio** — medido: o reflog da branch é apagado junto com ela, o do `HEAD` só registra se você esteve nela, e a única pista restante é prosa.

Daí o diário próprio da [SRS — undo](../srs/undo.md), sob o `--git-common-dir`. E daí um achado que reforça esta decisão: a rede é praticamente eterna para branch mergeada por ancestralidade, e fina para squashada ou rebaseada — ou seja, **mais fraca justamente onde a deleção era mais arriscada**.

## Relacionadas

- **[SRS — branches](../srs/branches.md)** — a regra de que a cláusula 1 nasceu, e a que a rede remota obedece.
- **[SRS — undo](../srs/undo.md)** — a implementação da prova de recuperação.
- **[ADR-002](002-dois-front-ends.md)** — a ordem `filtrar → Select → Delete`, generalizada aqui.
- **[ADR-005](005-gtr-ignore.md)** e **[ADR-007](007-soltar-branch-presa.md)** — as duas propõem operações que caem sob esta.
