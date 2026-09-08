# Contribuindo com o gitarias

Convenções de commit, histórico, código, teste e build deste repositório. A especificação de cada comando vive em [`docs/srs/`](docs/srs/); as decisões de arquitetura, em [`docs/adr/`](docs/adr/).

---

## 1. Commits

**Conventional Commits, em inglês.** Assunto no imperativo, máximo 50 caracteres. Corpo quebrado em 72 colunas.

### Tipos aceitos

| Tipo | Quando | Entra no `CHANGELOG` |
|---|---|---|
| `feat` | Funcionalidade nova, percebida por quem usa | Sim |
| `fix` | Correção de comportamento errado | Sim |
| `perf` | Ganho de desempenho sem mudança de comportamento | Sim |
| `refactor` | Muda estrutura preservando comportamento. Extração de pacote, inversão de dependência, mover lógica de lugar | Não |
| `test` | Adiciona ou corrige teste, incluindo fixture e helper de teste | Não |
| `docs` | Só documentação — README, este arquivo, especificação, comentário de API | Não |
| `style` | Formatação sem mudança de código (`gofmt`) | Não |
| `build` | Módulo, dependências, `go.mod` | Não |
| `ci` | Pipeline, workflow | Não |
| `chore` | Manutenção que não cai em nenhum dos acima. **Último recurso**, não guarda-chuva | Não |

**O critério do `CHANGELOG` é o tipo, não a ausência de `chore`.** Escolha o tipo que descreve a mudança; deixe o gerador filtrar. Um `refactor` marcado como `chore` não quebra o changelog, mas **apaga do histórico a informação de que aquilo foi refatoração** — e o histórico é o único registro que sobrevive.

### Demais regras

- **Cada commit compila.** Verificável com `gtr commits check <base> -- go build ./...`, que extrai a árvore de cada commit para um diretório temporário e roda o comando lá. É o que mantém o `git bisect` utilizável. Na prática **cada commit também passa a suíte** — teste vermelho num commit intermediário quebra o bisect tanto quanto compilação.
- **Commits em unidades lógicas.** Uma mudança que pode ser dividida em passos que compilam sozinhos deve ser dividida.
- **Autoria:** `Luiz Palma <83788360+LHPalma@users.noreply.github.com>`. O e-mail *noreply* vincula à conta sem expor o endereço real, o que importa porque o repositório vai ficar público.
- **Sem trailer de atribuição.** Nada de `Co-Authored-By`, `Signed-off-by` ou equivalente, em commit ou PR.

## 2. Histórico

- **Linear, sempre.** Fast-forward, squash ou rebase — **nunca merge commit**. `git log --merges` deve devolver vazio.
- Recomendado desmarcar *Allow merge commits* em Settings → General, para a regra ser do repositório e não de disciplina.
- **Nomes de branch são definidos pelo autor**, nunca inventados por ferramenta ou assistente.

## 3. Código

- **Um tipo por arquivo.** Mais granular que a doutrina de Go, que agrupa por coesão. Escolhido por navegabilidade. Exceção: o arquivo do tipo que concentra operações (`repo.go`) guarda os métodos junto do receiver.
- **Nomes completos, não abreviados.** `command` e não `c`, `output` e não `out`, `currentBranch` e não `current`. Receiver segue a mesma regra: `repo`, não `r`. Exceções toleradas onde o idioma já fixou a forma curta — o campo `Err` de um struct de resultado, o `args` que vem da assinatura do Cobra.
- **Português só no que o usuário lê.** Identificadores, nomes de pacote e mensagens de commit em inglês. Texto que sai no terminal, em português.
- **O domínio não escreve na tela nem lê do teclado.** Devolve dados; quem imprime é a apresentação. Inclui texto de exibição: use enum e traduza na apresentação.
- **DRY é sobre conhecimento, não sobre tokens.** Parâmetro repetido em várias chamadas não é duplicação; **a mesma decisão escrita em dois lugares é**. E abstração só se paga onde a feature futura vai usá-la.

### Comentário declara contrato, não história

Go não tem vocabulário de tags — nada de `@param`, `@return` ou `<summary>`. O *doc comment* é prosa colada à declaração, **aberta pelo nome do identificador**, em frases completas, e o `gofmt` a canoniza desde o Go 1.19.

**O que o comentário diz é o contrato:** o que o identificador faz, o que devolve e em que casos de borda. Nunca **como o código veio a ser** — nada que exija conhecer um item de roadmap, a intenção de quem escreveu ou um episódio de sessão para ser entendido. **Comentário que precisa da história do projeto para fazer sentido não é documentação do pacote.**

A razão de decisão mora na especificação, em `docs/srs/`, que é onde se revisita.

Comentário dentro de função é exceção, e serve para o que o código não consegue dizer sozinho: por que duas operações saem juntas, por que uma ordem importa, que armadilha uma guarda evita. Se o comentário só reafirma o que a linha faz, o problema é o nome.

**Documentar todo identificador exportado é doutrina de Go e está adiado por decisão explícita** — `internal/` não publica documentação, e a regra existe para consumidor externo, que não existe enquanto tudo for interno. Quando vier, o portão natural é lint, não disciplina.

## 4. Testes

| Elemento | Idioma | Forma |
|---|---|---|
| Nome da função de teste | Inglês | Doutrina de Go: nomeia o alvo (`TestParse`, `TestResolveBase`). O cenário vai no subteste. Função separada só quando a asserção é de natureza diferente — `TestDeleteNeverForcesWithoutPermission` |
| Nome do subteste | Português | **Sem acentos.** Vira parte do identificador e entra no `go test -run` |
| Mensagem de falha | Português | Com acentos. Nunca é filtrada, só lida |

O papel é o mesmo do `@DisplayName` de JUnit: o identificador é código, a descrição é prosa.

**Sobre o "sem acentos":** a regra não é remover o acento e seguir. Em português o acento às vezes distingue palavras — `é` vira `e`, `válida` vira `valida`. **Quando remover mudar a palavra, reescreva a frase.**

```text
"o segundo é o atual"                       → "marca o segundo"
"rev-parse falha e a lista continua válida" → "rev-parse falha sem invalidar a lista"
"repositorio bare nao tem HEAD"             ← ok, só grafia
```

### Fake de git

`internal/git/gittest` fornece um `Runner` **roteirizado por comando exato**. Ele **falha em comando não roteirizado** em vez de devolver vazio, para que uma chamada inesperada apareça como erro. A lista `Calls` permite asserção negativa — verificar que uma regra proibida (`branch -D` sem `--force`, leitura de ref remota, sondagem de branch protegida) nunca foi executada.

**As chaves dos mapas de resposta ficam soletradas literalmente.** Construí-las com a mesma lógica que o código de produção usa deixaria uma troca na ordem dos argumentos passar nos dois lados ao mesmo tempo.

### Cobertura não é evidência

**100% mede linha executada, não asserção feita.** Ao fechar um alvo, quebre o código de propósito e confirme que a suíte reclama; **mutação que sobrevive é buraco de teste, não ruído**.

Exceção: o **mutante equivalente** — mutação semanticamente idêntica ao original, que nenhum teste externo pode distinguir. Não deve ser "fechado" com asserção artificial; deve ser registrado na especificação da feature.

**A barra é 100%, e o repositório está em 99,2%** — medido com `go test ./... -coverpkg=./...`, que é o número que conta, porque a cobertura de um pacote vem em boa parte dos testes de quem o usa. O que falta está nomeado nas especificações de cada feature, e cai em três categorias:

| Categoria | Exemplos | O que fazer |
|---|---|---|
| **Fronteira** | `main.main`, o `run` do `git.CommandRunner` | Nada. É onde o processo toca o mundo, e a regra do projeto já os isenta |
| **Estrutural** | `doctor.ScratchVariable`, que tem um ramo por sistema operacional | Nada. Cobrir exigiria injetar o sistema numa função de duas linhas — costura de teste no código de produção |
| **Lacuna de verdade** | `ui.DescribeSection`, em 30,8% | Fechar. Um `switch` de doze ramos sem teste direto deixa passar troca de rótulo entre casos |

**Não há portão de cobertura no CI**, de propósito: um número no pipeline vira meta, e meta de cobertura se cumpre com teste que executa sem afirmar. A disciplina é a mutação, e ela não se automatiza.

### Teste de cenário precisa provar que o cenário existiu

Um cenário que verifica ausência — "o arquivo X não foi criado", "a branch Y não aparece" — **passa de graça quando a montagem falha silenciosamente**. Aconteceu duas vezes neste projeto:

1. Um cenário criava uma branch chamada `feat;touch OWNED`, nome que o git **recusa** porque ref não aceita espaço. Nenhuma branch era criada, e a asserção passava sem ter testado nada.
2. Um subteste chamado "commit-tree falhando" roteirizava, na verdade, o `cherry` falhando. Descoberto porque a cobertura ficou em 96,6% com exatamente aquelas linhas sem executar.

**Antes de afirmar ausência, verifique que a pré-condição foi montada.**

### Refactor precisa provar que não mudou comportamento

Commit `refactor:` afirma preservação de comportamento, e a afirmação precisa valer. O método usado neste projeto: construir o binário anterior a partir de `git archive <commit>` e comparar `stdout`, `stderr` e código de saída em cenários reais — **incluindo os caminhos de erro**, e conferindo antes que o `stderr` esperado não está vazio.

## 5. Build

```bash
CGO_ENABLED=0 go build -o gtr .
```

**`CGO_ENABLED=0` é obrigatório**, não otimização — detalhe em [ADR-001](docs/adr/001-go-e-binario-unico.md).

Antes de abrir PR: `gofmt -l .` vazio, `go vet ./...` limpo, `go test ./...` verde.

**Isso não é disciplina, é portão**: o CI roda os três a cada push e a cada PR, mais o build estático com asserção de `statically linked` e o cross-compile para darwin e windows.

### O que o portão não cobre

- **Qualquer coisa que dependa do `git` real** — a suíte inteira roda contra o fake, então uma mudança de formato entre versões do git passaria despercebida. A defesa é ler sempre de saída contratual, que é justamente a que não muda.
- O caminho que toca o mundo de fora: o `main`, e os runners de processo e de HTTP.

## 6. Dependabot

Semanal, para o módulo Go e para as actions, com **prefixo de commit por tipo** (`build` para dependência, `ci` para workflow) em vez de `chore`.

Minor e patch de actions agrupados num PR só; **bump de Go fica um por PR**, para que cada commit continue compilando sozinho.

`allow: dependency-type: all` — o `cobra` é a única dependência direta, então as regras padrão deixariam o updater praticamente ocioso.
