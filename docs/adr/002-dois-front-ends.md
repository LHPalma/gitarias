---
titulo: ADR-002 — Dois front-ends e o contrato Selector
data: 2026-08-05
status: entregue
escopo: internal/ui, tui/, cmd/
supersede: o modelo tudo-ou-nada do [y/N] como única forma de seleção
---

# ADR-002 — Dois front-ends e o contrato Selector

- **Data:** 2026-08-05
- **Status:** **entregue** — `ui.Selector`, o `tui/` e o `gtr branches --clean --interactive` estão na `main`
- **Escopo:** `internal/ui`, `tui/`, `cmd/`
- **Supersede:** o modelo tudo-ou-nada do `[y/N]` como única forma de seleção

---

## Contexto

A ferramenta terá **duas apresentações convivendo**: a CLI de texto atual e um front interativo acionado por flag.

**É assumidamente mais estrutura do que o tamanho do projeto exige.** O objetivo declarado é aprender a construir a separação — e isso está registrado de propósito, para que ninguém leia esta decisão como economia.

O caso de uso concreto é escolher **quais** branches apagar, em vez de tudo-ou-nada.

### Estrutura alvo

```text
internal/git/        execução de git                        ENTREGUE
internal/branch/     domínio: encontra, filtra e deleta     ENTREGUE
internal/worktree/   domínio: lista working trees           ENTREGUE
cmd/                 front de texto                         ENTREGUE
internal/ui/         vocabulário compartilhado              ENTREGUE
                     o contrato Selector                    PENDENTE
tui/                 front interativo (Bubble Tea)          PENDENTE
```

`internal/` tem significado especial em Go: o compilador proíbe que projetos externos importem o que está lá dentro. É encapsulamento no nível de diretório.

### Os quatro obstáculos, e como cada um foi resolvido

**A lógica imprimia de dentro do laço de deleção e lia o `stdin` diretamente.** Uma TUI controla a tela inteira: qualquer escrita solta borra o desenho, e a disputa pelo `stdin` quebra o tratamento de teclas. Resolvido quando o `Delete` passou a devolver `[]DeleteResult`, e a regra de o domínio não escrever na tela foi fixada na [SRS — branches](../srs/branches.md).

**O `cmd` falava direto com o processo e construía o `git.CommandRunner` dentro de cada comando.** Nenhum front alternativo tem por onde entrar num comando que constrói as próprias dependências. Resolvido pela **[ADR-003](003-injecao-de-dependencia.md)**.

**O vocabulário de exibição estava preso em funções privadas do `cmd`**, fora do alcance do `tui/`. Sem mover, a TUI nasceria duplicando "squashada" e "rebaseada". O `internal/ui` previsto aqui existe, com os tradutores exportados. Falta só o contrato.

**O `cmd` matava o processo com `os.Exit`.** Hoje devolve o código de saída, e quem encerra é o `main` — o que a TUI vai querer também: reportar um resultado em vez de derrubar o programa por baixo de si mesma.

## Decisão

**A abstração fica um nível acima do `[y/N]`:**

```go
type Selector interface {
    Select(candidates []branch.Branch) ([]branch.Branch, error)
}
```

- **Front de texto:** pergunta `[y/N]`; devolve todas as candidatas ou nenhuma.
- **Front TUI:** lista com checkbox; devolve apenas as marcadas.

A interface é `Select` e não `Confirm` de propósito: `Confirm` prenderia a TUI ao modelo mais pobre.

Go satisfaz interfaces implicitamente — não existe `implements`. O pacote `tui` cumpre o contrato sem precisar importar quem o declarou, o que inverte a dependência sem cerimônia.

### A ordem das operações faz parte do contrato

Tratar o tudo-ou-nada como **caso degenerado** da multisseleção deixou de valer quando o `--force` entrou: o front de texto **não devolve mais "todas ou nenhuma"**, devolve *o subconjunto autorizado*.

A lista deixou de ser homogênea. Depois da detecção de equivalência ela tem duas classes com permissões diferentes: mergeadas por ancestralidade, que o `-d` apaga, e equivalentes, que exigem `--force`.

```text
filtrar por autorização (--force)  →  Select  →  Delete
```

**`--force` é autorização; `Select` é quais.** Separadas de propósito. Se as duas se misturarem, a TUI herda uma pergunta sem resposta boa: o que fazer quando alguém marca uma branch que a autorização não cobre — esconder, mostrar apagada, deixar marcar e falhar no fim? **Mantendo o filtro antes do `Select`, a TUI só recebe o que já pode ser apagado, e a pergunta não chega a existir.**

### A flag

**`-i` / `--interactive`.** Venceu `--ui` e `--tui` por dois motivos: segue convenção que já existe na cabeça de quem usa terminal (`git add -i`, `docker run -i`), e **não amarra o nome à tecnologia escolhida** — se o Bubble Tea for trocado, `--tui` passaria a descrever a implementação em vez do que o usuário pede.

### Detecção de TTY

TUI não é pipeável. A ferramenta verifica `term.IsTerminal` e, quando não houver terminal:

- **falha** se a flag interativa foi pedida explicitamente — silêncio esconde bug;
- **cai para texto** se a escolha veio de detecção automática.

**Ressalva prática, não resolvida:** `term.IsTerminal` recebe um descritor de arquivo, e a apresentação escreve num `io.Writer` vindo do Cobra — abstração que **não tem** descritor. Vai exigir type assertion para `*os.File`, tratando a falha como "não é terminal", que é a resposta certa para um `bytes.Buffer` de teste. É a injeção cobrando um preço pequeno: em troca de testabilidade, perde-se o acesso direto ao `os.Stdout`.

**Dependências diretas novas**, as primeiras desde o cobra: `golang.org/x/term`, `bubbletea`, `bubbles` e `lipgloss`.

### Bibliotecas avaliadas

| Lib | Papel |
|---|---|
| `lipgloss` | Só estilo: cor, borda, alinhamento. Não muda a lógica. |
| `huh` | Formulários e prompts prontos. Substituiria o `[y/N]` por um seletor. |
| `bubbletea` | Framework interativo completo, arquitetura Elm (`Init` / `Update` / `View`). |
| `bubbles` | Componentes prontos sobre o `bubbletea`: `list` com filtro, `table`, `textinput`, `viewport`, `paginator`. É o que entrega listagem rica — o `bubbletea` sozinho é só a arquitetura. |

**Escolhidos: `bubbletea` + `bubbles` + `lipgloss`.**

O motivo é o que vem depois, não a seleção de branches. Para escolher branches, um formulário bastaria. Mas as telas de métrica — commits por autor e por dia — e a saída em `--format` querem tabela e listagem com filtro, que é outro problema; e a primeira tela que não for formulário obrigaria a reescrever o `tui/` inteiro.

Registrado explicitamente para não ser relido como precaução: **não se escolheu a maior para "não travar depois"**. Travar não era risco, e a reversibilidade nunca dependeu da lib — depende do `Selector`.

### Cobertura do `tui/` — sem exceção

O `tui/` **não** vira exceção à barra de cobertura. Vale a regra que o projeto já aplica: cobre-se tudo, menos a função que toca o mundo de fora — a mesma categoria de `git.CommandRunner.Run` e `main.main`.

É viável porque a arquitetura Elm ajuda mais do que atrapalha: **`Update` é função pura**, `(modelo, mensagem) → (modelo, comando)`, testável sem terminal nenhum. O que sobra descoberto é a chamada a `tea.NewProgram(...).Run()`, que é o boundary. E o `Selector` implementado pelo `tui` é lógica comum — mapear a seleção de volta para `[]branch.Branch` — com teste normal.

**Armadilha registrada:** não asserte sobre a saída estilizada do `View`. O `lipgloss` emite ANSI, e o resultado depende de perfil de cor e largura de terminal — teste que quebra por motivo errado. Asserte sobre o estado do modelo depois do `Update`, e sobre conteúdo sem estilo.

Fazer do `tui/` uma exceção seria a primeira vez que o projeto baixa a própria barra, e o custo apareceria noutro lugar: **se a camada de apresentação ficar difícil de testar, o sintoma é lógica no lugar errado** — que é o que o contrato e a regra do domínio que não imprime existem para impedir.

## Nota de entrega

O `-i`/`--interactive` chegou só no `branches --clean`, exatamente o caso de uso desta ADR. A lista começa com tudo marcado — `enter` reproduz o efeito do `[y/N]` antigo sem exigir que quem usa marque uma a uma; `a`/`n` marcam ou desmarcam todas de uma vez, e `espaço` alterna uma.

**`bubbles` não entrou junto.** A tela de checkbox não usa lista com filtro nem tabela — só `bubbletea` e `lipgloss` são importados agora. `bubbles` fica para quando a primeira tela de tabela ou lista (`stats`, `changelog`) precisar dele de verdade; declará-lo sem uso violaria a mesma regra que a ADR já aplica a outra escolha (§ "Bibliotecas avaliadas"): a decisão descreve o que vem depois, mas o `go.mod` só ganha o que o código importa.

**A cobertura do `tui/` fechou em 89,8%.** O único descoberto é `BranchSelector.Select`, na linha exata que chama `tea.NewProgram(...).Run()` — o boundary que a seção "Cobertura do `tui/`" já previa. `newModel`, `Update`, `View` e `selected` estão 100% cobertos por teste direto, sem terminal.

## Alternativas consideradas

**Escrever o contrato agora, antes da TUI.** Rejeitada, e por razão empírica: **esta interface já torceu uma vez sozinha.** Ela foi desenhada antes da detecção de equivalência e passou a descrever um modelo que não existe mais. Interface com uma implementação só, desenhada antes de a segunda existir, é onde abstração nasce torta. O contrato sai junto com a TUI.

**`Confirm(candidates) (bool, error)` em vez de `Select`.** Mais simples e suficiente para o front de texto. Rejeitada: prende a TUI ao modelo tudo-ou-nada, que é exatamente o que a seleção interativa existe para superar.

**`huh` no lugar do framework.** O caso de uso é literalmente um formulário de múltipla escolha, e um `huh.NewMultiSelect` resolveria em poucas dezenas de linhas. Rejeitada pelo que vem depois, não pela primeira tela. Vale registrar que **`huh` é construída sobre o `bubbletea`** — um formulário `huh` é um `tea.Model` e pode ser embutido depois, se alguma tela for só formulário. As duas nunca foram caminhos concorrentes.

**Um front só, sempre interativo.** Elimina a abstração inteira. Rejeitada: quebra a saída pipeável — `gtr branches | grep feat` deixaria de funcionar, e isso é metade do valor de uma CLI.

**Deixar cada comando decidir seu front, sem interface comum.** Rejeitada: o `stats` e o `changelog` vão querer a mesma seleção, e a terceira cópia é onde a divergência começa.

## Consequências

**Positivas**

- O tudo-ou-nada e a multisseleção deixam de ser modelos concorrentes.
- O `cmd` e o `tui` passam a ser intercambiáveis pelo ponto de entrada que a **[ADR-003](003-injecao-de-dependencia.md)** criou.
- A ordem `filtrar → Select → Delete` mantém a autorização fora da interface de seleção.

**Negativas**

- Estrutura acima do necessário para o tamanho atual — assumido explicitamente.
- Quatro dependências diretas novas de uma vez, num projeto que tinha só o cobra.
- A detecção de TTY fica com um type assertion que só existe por causa da injeção.
- Mais código para a primeira tela do que um formulário exigiria.

**Neutras**

- O front de texto continua sendo o padrão; nada muda para quem não usar `-i`.

## Relacionadas

- **[ADR-003](003-injecao-de-dependencia.md)** — pré-requisito entregue; é por onde o segundo front entra.
- **[SRS — equivalência por conteúdo](../srs/equivalencia.md)** — a feature que envelheceu o contrato descrito aqui.
- O `internal/ui` é a casa dos rótulos, e o ponto de partida da internacionalização.
