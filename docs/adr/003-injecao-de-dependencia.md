---
titulo: ADR-003 — Injeção de dependência no cmd, sem container
data: 2026-08-05
status: entregue
escopo: cmd/
supersede: comandos declarados como variáveis de pacote com init(), e flags como variáveis globais
---

# ADR-003 — Injeção de dependência no cmd, sem container

- **Data:** 2026-08-05
- **Status:** **entregue**
- **Escopo:** `cmd/`
- **Supersede:** comandos declarados como variáveis de pacote com `init()`, e flags como variáveis globais

---

## Contexto

### Inversão já existia; faltava usá-la

Distinção que vale registrar porque os dois termos são trocados o tempo todo:

| Conceito | O que é | Estado |
|---|---|---|
| **Inversão** de dependência | O domínio depende de abstração em vez de concreto | **Feita** — `git.Runner` é interface desde o começo |
| **Injeção** de dependência | Receber a dependência de fora em vez de construí-la dentro | **Faltava**, e só no `cmd` |

O sintoma era a primeira linha de cada comando:

```go
repo := branch.NewRepo(git.CommandRunner{})
```

**A porta já existia; o `cmd` entrava pela parede.** Por isso `runBranches`, `runWorktrees` e `Execute` estavam a 0% enquanto os dois domínios estavam a 100% — exercitá-los exigia repositório git de verdade em disco, que é exatamente o que a interface tinha eliminado.

### O segundo problema, no mesmo lugar

As flags eram **variáveis de pacote**:

```go
var branchesClean bool
var branchesBase  string
var branchesForce bool
```

Estado global mutável compartilhado por toda execução. Um teste que ligasse `--force` contaminaria o próximo, e a ordem dos testes passaria a importar — o que torna o comando difícil de testar **mesmo com o runner injetado**.

## Decisão

**Comandos passam a ser construídos em vez de declarados**, recebendo o `Runner` por parâmetro.

```go
func Execute() {
    command := newRootCommand(git.CommandRunner{})
    if err := command.Execute(); err != nil {
        fmt.Fprintln(command.ErrOrStderr(), "erro:", err)
        os.Exit(1)
    }
}

func newRootCommand(runner git.Runner) *cobra.Command {
    command.AddCommand(newBranchesCommand(runner))
    command.AddCommand(newWorktreesCommand(runner))
}
```

O `CommandRunner` real entra em **um lugar só** e desce por parâmetro. As flags viram locais capturados pelo closure do `RunE`, então cada comando construído tem as suas.

Os streams vêm do próprio Cobra — `OutOrStdout`, `ErrOrStderr`, `InOrStdin` — que resolvem para o processo por padrão. Nada a configurar em produção; `SetOut`, `SetErr` e `SetIn` trocam nos testes.

**Nenhuma abstração nova nasceu.** A diferença entre testável e intestável foi mover uma dependência do corpo para a assinatura.

### Por que nenhum container de DI

Em Spring ou .NET um container monta o grafo de objetos em runtime, por reflexão sobre anotações. Existe porque montar o grafo à mão dói: hierarquias profundas, muitos objetos, escopos por requisição e por sessão.

**Go é deliberadamente o oposto: wiring manual e explícito, concentrado no ponto de entrada.** O grafo inteiro do `gtr` cabe numa linha, sem registro, sem anotação, sem reflexão — e o compilador verifica tudo.

Três traços da linguagem sustentam isso:

| Traço | Efeito |
|---|---|
| Interfaces satisfeitas implicitamente | Não existe `implements`. O `CommandRunner` nunca menciona `git.Runner`; só tem o método certo. É por isso que o `gittest.Runner` funciona sem importar nada do `branch` |
| Interfaces pequenas, declaradas por quem consome | A do projeto tem **um** método. Implementar um método é barato; é o que torna o fake trivial |
| Struct de opções em vez de setters | `branchesOptions` passado por valor, no lugar de campos configurados por injeção |

**Containers existem em Go** — `google/wire` (codegen em tempo de compilação), `uber-go/dig` (reflexão em runtime), `uber-go/fx` (`dig` mais ciclo de vida). Só se pagam com grafo grande de verdade: dezenas de serviços, ordem de shutdown, escopos.

**Decisão: nenhum container, agora nem depois.** Para uma CLI com um runner e dois comandos, o `wire` geraria mais código do que a linha que já existe. Se o grafo crescer a ponto de doer, o problema é o grafo.

### Onde os rótulos moram

Os tradutores de enum eram privados do `cmd`:

```text
cmd/branches.go:   describeMerge, describeSource
cmd/worktrees.go:  describeCheckout, describeState
```

A **[ADR-002](002-dois-front-ends.md)** põe o front interativo em `tui/`, pacote separado na raiz. Ele **não alcança função privada do `cmd`**, e importar `cmd` a partir de `tui` seria dependência ao contrário. Então ou a TUI duplica `"squashada"` e `"rebaseada"`, ou os rótulos ganham casa antes.

Destino natural: o `internal/ui` que a ADR-002 já prevê — o mesmo pacote do contrato guardando o vocabulário compartilhado pelos dois fronts. **É também o ponto de costura da internacionalização**, então não é trabalho antecipado à toa.

Os quatro viraram `ui.DescribeMerge`, `ui.DescribeSource`, `ui.DescribeCheckout` e `ui.DescribeState`, com `withReason` e `shortHead` privados junto. O pacote importa `branch` e `worktree`; nenhum dos dois o importa, e eles continuam sem se conhecer. Os testes foram junto sem alteração, e a saída foi conferida byte a byte contra o binário anterior em nove invocações.

O contrato `Selector` **não** veio junto, de propósito: a ADR-002 o mantém atrelado à chegada da TUI.

Foi o que motivou a reescrita da regra do domínio que não imprime: dizer que texto de exibição "mora no `cmd`" era exato com uma apresentação só, e vira ordem de duplicar com duas.

## Alternativas consideradas

**Variável de pacote trocável pelo teste** (`var newRunner = func() git.Runner { ... }`). É o atalho comum em Go. Rejeitada: introduz estado global mutável, que é exatamente metade do problema sendo corrigido — testes não poderiam rodar em paralelo e a ordem voltaria a importar.

**Struct de aplicação carregando as dependências**, com os comandos como métodos. Chega ao mesmo resultado. Rejeitada por peso: para dois comandos, o construtor por comando é menos cerimônia e mais legível.

**`google/wire` para gerar o wiring.** Rejeitada: o grafo tem três nós. O codegen produziria mais linhas do que a chamada única que ele substituiria, e adicionaria um passo ao build.

**Não fazer nada e registrar as três funções como descobertas.** Opção honesta e considerada. Rejeitada porque **duas razões independentes pediam a mesma mudança** — a cobertura e o segundo front —, e isso é o melhor sinal que existe de que ela é a certa.

## Consequências

**Positivas**

- O `cmd` sai de 54,3% para **100%**. O único descoberto era o `Execute`, que chamava `os.Exit` — e ele deixou de existir: o `cmd` devolve o código de saída e quem chama `os.Exit` é o `main`, a mesma correção de altitude desta decisão, um nível acima. O teste por subprocesso foi tentado antes e medido: o filho executa a função, mas os contadores morrem com ele e o perfil do pai não se move.
- Os testes dirigem os comandos Cobra de verdade contra um git roteirizado — ordem das chamadas, flags e retornos antecipados passam a ser verificáveis.
- Flags deixam de ser estado compartilhado. `TestCommandsNeverShareFlagState` pega a regressão.
- O ponto de entrada do segundo front existe: `newBranchesCommand(runner)`.

**Negativas**

- Perde-se o acesso direto ao `os.Stdout`, o que cobra um type assertion na detecção de TTY (**[ADR-002](002-dois-front-ends.md)**).
- O corpo dos construtores fica um pouco mais longo que a declaração de variável que substituiu.

**Neutras**

- Comportamento idêntico, verificado byte a byte contra o binário anterior em nove cenários — incluindo `--help` e comando inválido, que passam pelo Cobra e não pelo código do projeto.
- Os `init()` desaparecem; o registro dos subcomandos passa a ser explícito no `newRootCommand`.

## Relacionadas

- **[ADR-002](002-dois-front-ends.md)** — a segunda razão independente que pedia esta mudança.
- **[ADR-001](001-go-e-binario-unico.md)** — mesma natureza: o requisito existia, faltava o mecanismo cumpri-lo.
- **[SRS — branches](../srs/branches.md)** — o `deletable` extraído aqui é a decisão de segurança que passou a ter teste.
