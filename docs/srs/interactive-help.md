---
titulo: SRS — menu interativo de ajuda
data: 2026-09-12
status: entregue
comando: gtr (sem subcomando)
pacotes:
  - cmd/root.go
  - cmd/help_menu.go
  - tui
fonte_externa: nenhuma
---

# SRS — menu interativo de ajuda

- **Data:** 2026-09-12
- **Feature:** `gtr` sem subcomando, com terminal, abre um menu navegável em vez do help estático do Cobra
- **Status:** **entregue** — na `main`
- **Fonte externa:** nenhuma

---

## 1. Introdução

### 1.1 Propósito

Hoje `gtr` sem argumento imprime o help padrão do Cobra: uma lista estática de comandos e uma linha de descrição cada. Esta feature troca essa lista, quando há terminal de verdade, por um menu navegável — setas escolhem o comando, `enter` mostra o help completo dele (o mesmo que `gtr <comando> --help` imprimiria), `esc` volta pra lista.

É o segundo front interativo do projeto, depois do `branches --clean -i` da **ADR-002**, e reaproveita a mesma decisão de arquitetura: `tui/` guarda o modelo Elm (`Update` puro), `cmd/` decide quando usá-lo.

### 1.2 Escopo

**Entregue:** o menu em si (`tui.HelpMenu` + `tui.menuModel`), o `RunE` do comando raiz que aciona o menu só na invocação sem subcomando, a detecção de terminal (cai pro help padrão sem um), e a captura do texto de ajuda de cada subcomando sem vazar escrita direta pro terminal por baixo da TUI.

**Fora de escopo:** qualquer flag explícita para isto (`-i` continua exclusivo do `branches --clean`) — aqui a interatividade é **detecção automática** por terminal, não pedido explícito, e por isso a ausência de terminal cai em silêncio para o texto (ADR-002, seção "Detecção de TTY").

---

## 2. Descrição geral

```text
tui/menu_item.go     MenuItem{Name, Short}
tui/help_menu.go     HelpMenu — Run(items, detail)
tui/menu_model.go    menuModel — Update puro, duas telas: lista e detalhe

cmd/root.go          RunE do comando raiz aciona runInteractiveHelp
cmd/help_menu.go     runInteractiveHelp, menuItems, helpText
```

### 2.1 Fluxo

1. `gtr` é chamado sem subcomando — dispara o `RunE` do comando raiz.
2. Sem terminal (`isTerminal` nega): `command.Help()` — o help padrão do Cobra, sem diferença do comportamento anterior.
3. Com terminal: monta a lista de itens a partir de `command.Commands()`, abre o `tui.HelpMenu`.
4. Setas navegam; `enter` chama `helpText` no comando selecionado e mostra o resultado; `esc` no detalhe volta pra lista; `esc`/`q`/`ctrl+c` na lista encerra.

---

## 3. Requisitos funcionais

### RF-01 — Menu ao chamar `gtr` sem subcomando, com terminal

Lista todo comando que apareceria em "Available Commands" no help padrão — mesmo filtro do próprio Cobra: oculta `Hidden` e `Deprecated`, mantém o `help` auto-gerado.

### RF-02 — Mostrar o help completo do comando escolhido

`enter` sobre um item mostra o que `gtr <comando> --help` imprimiria: descrição, uso e flags. `esc` volta pra lista sem sair do programa.

### RF-03 — Cair pro help padrão sem terminal

Sem terminal de verdade — pipe, script, CI —, `gtr` sem argumento continua exatamente como antes: o help estático do Cobra, sem menu, sem erro. Diferente do `--interactive` do `branches`, aqui não há flag explícita pedindo o modo interativo, então a ADR-002 manda cair em silêncio, não falhar.

### RF-04 — `gtr <comando inválido>` continua um erro

`legacyArgs` do Cobra já recusa argumento que não bate com nenhum subcomando antes de qualquer `Run` rodar — o `RunE` do comando raiz nunca chega a ver esse caso, e a mensagem `unknown command "x" for "gtr"` não muda.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **O filtro do menu espelha o do template padrão do Cobra**, não reinventa um próprio: `subcommand.IsAvailableCommand() \|\| subcommand.Name() == "help"` — a mesma condição do `UsageTemplate` interno. Divergir aqui faria o menu mostrar (ou esconder) algo que o help estático não concorda. |
| **RN-02** | **Capturar o help de um comando não pode escrever no terminal por baixo da TUI.** `helpText` redireciona `SetOut` do subcomando pra um `bytes.Buffer` antes de chamar `Help()`, e desfaz com `SetOut(nil)` depois — o mesmo idioma que o próprio `Command.UsageString()` usa internamente. |
| **RN-03** | **Interatividade aqui é detecção automática, nunca pedido explícito.** Por isso a ausência de terminal cai em silêncio pro help de texto — ao contrário do `--interactive` de `branches`, que falha sem terminal por ter sido pedido explicitamente. As duas regras vêm da mesma ADR-002 e não se contradizem: cada uma resolve o caso que descreve. |

---

## 5. Interface

```console
$ gtr
Comandos do gtr:

> branches               Lista branches locais já mergeadas na branch base
  changelog               Gera o CHANGELOG.md a partir do histórico...
  ...

↑/↓ move · enter mostra o help · esc/q sai
```

Sem terminal, a saída é idêntica à de antes desta feature: o help padrão do Cobra.

---

## 6. Testes

### 6.1 Cobertura automatizada

`tui/menuModel` — `Update` e `View` cobertos a 100%, testados sem terminal (o `Update` é função pura). `HelpMenu.Run` fica descoberto pelo mesmo motivo que `BranchSelector.Select`: é a chamada a `tea.NewProgram(...).Run()`, o boundary que a ADR-002 já reconhece.

`cmd/help_menu.go` — `menuItems` e `helpText` cobertos a 100%, incluindo o filtro de comandos ocultos e o reset do `SetOut` depois de capturar. `runInteractiveHelp` fica em 33,3%: só o ramo sem terminal é testável sem um de verdade — o ramo que abre a TUI é o mesmo tipo de boundary.

### 6.2 Cenários

| # | Cenário | Esperado |
|---|---|---|
| 1 | `gtr` sem terminal (saída é `*bytes.Buffer` de teste) | Help padrão do Cobra, `Available Commands:` presente |
| 2 | `menuItems` sobre a árvore real de comandos | `favorite-band` (Hidden) fora, `branches` e `help` dentro |
| 3 | `helpText(branches)` | Mesmo texto de `gtr branches --help` |
| 4 | `helpText` chamado e depois `gtr branches --help` de novo | Continua funcionando — o `SetOut(nil)` não deixa resíduo |
| 5 | `menuModel`: setas, `enter`, `esc` na lista e no detalhe, `q`/`ctrl+c` nos dois lugares | Cobertos em `tui/menu_model_test.go`, sem terminal nenhum |

---

## 7. Rastreabilidade

| Commit | Entrega |
|---|---|
| `85b10ae` | `tui.HelpMenu` e `menuModel` — o menu em si, sem o `cmd` ainda enxergar |
| `ebdef25` | `RunE` do comando raiz aciona o menu; `menuItems`, `helpText` |

---

## 8. Follow-ups conhecidos

- **Sem paginação ou filtro no menu.** Com ~20 comandos cabe numa tela; se a lista crescer muito, é o momento de reconsiderar `bubbles` (ADR-002, nota de entrega).
- **`helpText` reconstrói o texto a cada `enter`**, mesmo sobre o mesmo item — sem cache. Não é gargalo perceptível (é só formatação de string), mas fica registrado.
