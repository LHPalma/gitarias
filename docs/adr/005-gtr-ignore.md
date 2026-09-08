---
titulo: "ADR-005 — gtr ignore: ler e editar o .gitignore"
data: 2026-08-05
status: entregue
escopo: gtr ignore list, gtr ignore add
supersede: o comando de topo gtr ignored
---

# ADR-005 — gtr ignore: ler e editar o .gitignore

- **Data:** 2026-08-05
- **Status:** **entregue** — `list` e `add`, com as duas decisões que ficavam em aberto resolvidas na [SRS — ignore](../srs/ignore.md)
- **Escopo:** `gtr ignore list`, `gtr ignore add`
- **Supersede:** o comando de topo `gtr ignored`, especificado antes como comando próprio

---

## Contexto

Duas perguntas que o git responde mal:

1. **"O que está sendo ignorado aqui, e por qual regra?"** — exige compor dois comandos de plumbing.
2. **"Adiciona esse padrão ao `.gitignore`"** — trivial de fazer errado, e o git não avisa.

É também a primeira feature do projeto que **escreve arquivo** no repositório.

### Por que subcomando, e não dois comandos de topo

O desenho anterior previa `gtr ignored` como comando de topo, ao lado de um futuro `gtr ignore`. **`ignore` e `ignored` diferem por uma letra.** Digitar errado é irrelevante — o problema é ler o help e não distinguir qual dos dois escreve e qual só lê.

Com subcomando, a leitura fica na palavra que importa (`add` ou `list`) e não no sufixo.

## Decisão

```text
gtr ignore add <padrão>   insere uma entrada
gtr ignore list           lista o que está ignorado, com a regra que ignorou
```

Efeito colateral desejável: abre espaço para `gtr ignore why <caminho>` — o `check-ignore -v` de um caminho só. Não especificado ainda.

### `list` — a pipeline

```bash
git ls-files --others --ignored --exclude-standard --directory --no-empty-directory -z \
  | git check-ignore -z --stdin -v
```

Quatro campos por registro — arquivo da regra, linha, padrão, caminho — que são as colunas do CSV saindo prontas:

```text
.gitignore|1|node_modules/|node_modules/
.gitignore|2|*.log|app.log
.gitignore|4|relatório *.csv|relatório 2026.csv
```

**Colapsado por padrão.** Sem `--directory`, um repositório com `node_modules` cospe centenas de milhares de linhas. `--expand` lista arquivo a arquivo. O `--no-empty-directory` acompanha, senão o git lista o diretório *e* os arquivos dentro dele.

**`-z` é obrigatório.** O git **cita** caminhos com acento ou caractere especial: `relatório 2026.csv` sai como `"relat\303\263rio 2026.csv"`. Mesma família de não confiar em separador que pode aparecer no dado.

**Duas sutilezas para a documentação:**

- `--exclude-standard` **não é só o `.gitignore`**: inclui `.git/info/exclude` e o `core.excludesFile` global. Por isso a coluna "arquivo da regra" importa — ela distingue "o time ignorou" de "sua máquina ignorou".
- Arquivo **já rastreado** que casa com uma regra não aparece; o git continua rastreando quem já estava dentro. É o comportamento desejado, mas quem procura "por que esse arquivo não sumiu" precisa saber.

### `add` — o destino é escolha de quem roda

**O `gtr` não elege destino em silêncio.** Os três que o `--exclude-standard` enxerga são expostos:

| Destino | Significado |
|---|---|
| `<raiz>/.gitignore` | Convenção do repositório, revisável em PR, vale para todos |
| `.git/info/exclude` (`--local`) | Preferência sua naquele clone; ninguém mais vê |
| `core.excludesFile` (`--global`) | Sua, em todos os repositórios da máquina |

Mesma postura da **[ADR-004](004-configuracao-e-formatos.md)**, onde quem decide se a configuração é do time ou pessoal é o git e não o `gtr`. Aqui a decisão é da pessoa, explicitamente, no momento da chamada — e é a mesma distinção que a coluna "arquivo da regra" já sabe exibir do lado da leitura.

### As seis armadilhas do `add`

| # | Armadilha | Tratamento |
|---|---|---|
| 1 | **Escape do padrão.** `#temp` gravado cru vira comentário e não ignora nada; `!importante` vira negação e **des-ignora**; `build ` com espaço final perde o espaço | Escapar `#`, `!` e espaço final na gravação — não tratar dado do usuário como inerte |
| 2 | **Padrão que não casa com nada é typo silencioso.** `node_module/` sem o `s` grava com sucesso e não ignora coisa alguma | Depois de gravar, verificar se a regra nova pegou algum caminho; **avisar** quando não pegou. Não é erro — pode ser regra preventiva |
| 3 | **Arquivo já rastreado não some.** É a confusão nº 1 do `.gitignore` | Se o padrão casa com arquivo rastreado, dizer que a regra é inócua até um `git rm --cached`. **Nunca executar o `rm` sozinho** |
| 4 | **Regra redundante.** Com `*.log` presente, adicionar `app.log` é lixo | `check-ignore` **antes** de gravar; se já estava coberto, informar qual regra cobre e não gravar |
| 5 | **Higiene de escrita.** Arquivo sem newline final faz o append colar na última linha e corromper a regra anterior; o arquivo pode nem existir | Garantir newline antes de anexar; criar o arquivo quando ausente |
| 6 | **Entrada duplicada literal** | Comparar com as linhas existentes antes de gravar |

### Ordem das verificações

1. Resolver o destino, pela flag de quem roda.
2. `check-ignore` no padrão: já está coberto? → armadilha 4, não grava.
3. Ler o arquivo de destino: linha idêntica já existe? → armadilha 6, não grava.
4. Escapar o padrão (armadilha 1) e anexar, garantindo newline (armadilha 5).
5. `check-ignore` de novo: a regra pegou alguma coisa? → armadilha 2.
6. Algum caminho que ela casa já está rastreado? → armadilha 3.

Os passos 5 e 6 são **avisos depois de gravar**, não condições de gravação. Gravar e avisar é diferente de recusar: uma regra preventiva, escrita antes de o arquivo existir, é uso legítimo.

### Duas implicações arquiteturais

**1. O `Runner` precisa ser estendido.** A segunda metade da pipeline do `list` lê do **stdin**, e o port era `Run(args ...string) (string, error)`, sem como alimentá-lo. Passar os caminhos como argumento estoura o limite de argv em repositório grande.

Foi a primeira feature a exigir mexer no contrato que toda a suíte usa, e a saída não foi nenhuma das duas opções óbvias — nem método novo na interface, nem interface separada. A interface `Runner` ficou **intocada**, com um método; a capacidade foi para as implementações (`RunWithInput` no `CommandRunner` e no `gittest.Runner`), e **cada consumidor declara a interface que precisa**, que é o que a **[ADR-003](003-injecao-de-dependencia.md)** já defende. Quem só usa `Run` continua com um fake de um método.

**Armadilha que este desenho não previa, e a implementação achou:** o `check-ignore` **sai com código 1 quando nada é ignorado**, e isso é resposta, não falha. O `CommandRunner` usava `Output()`, que trata qualquer status não-zero como erro e descarta o código — "nada ignorado" seria indistinguível de repositório quebrado. Daí o `git.ExitError`, alcançável por `errors.As`, com a mensagem preservada para o erro continuar acionável.

**2. Nenhum port de filesystem.** O `Runner` existe porque git é processo externo, com saída para parsear e caro de montar em teste. Arquivo não tem esse problema: o domínio recebe o caminho de destino como parâmetro e usa `os` direto, e o teste passa um `t.TempDir()`.

Inventar um port aqui seria **simetria sem ganho** — e a **[ADR-002](002-dois-front-ends.md)** já assume estrutura acima do necessário por decisão consciente, **o que não é licença para repetir o padrão onde ele não paga**.

A regra do domínio que não escreve na tela continua valendo e não é contrariada: ela proíbe **escrever na tela e ler do teclado**, não escrever em arquivo. O que o `add` devolve é resultado — o que foi gravado, onde, e os avisos das armadilhas 2, 3 e 4 —, e quem imprime segue sendo a apresentação.

## Alternativas consideradas

**Manter `gtr ignored` como comando de topo.** Rejeitada pela colisão de leitura com um futuro `gtr ignore`.

**Executar `git rm --cached` quando a regra for inócua** (armadilha 3). Rejeitada: é operação destrutiva disparada por um comando que o usuário chamou para *adicionar uma linha de texto*. Avisar é o limite.

**Recusar a gravação quando o padrão não casa com nada** (armadilha 2). Rejeitada: regra preventiva é uso legítimo — escrever `dist/` antes de existir `dist/` é o caso comum.

**Criar port de filesystem por simetria com o `Runner`.** Rejeitada — ver acima.

**Passar os caminhos como argumento em vez de estender o port.** Rejeitada: estoura o limite de argv em repositório grande, que é exatamente o caso onde a feature importa.

## Consequências

**Positivas**

- Responde "quem ignorou este arquivo" sem compor plumbing à mão.
- As quatro colunas já saem no formato que o `--format csv` da **[ADR-004](004-configuracao-e-formatos.md)** quer.
- As seis armadilhas viram avisos em vez de surpresas.

**Negativas**

- **Primeira escrita em arquivo do repositório** — a ferramenta deixa de ser somente-leitura sobre o working tree.
- Força a extensão do `git.Runner`, que toda a suíte de testes usa.

**Neutras**

- O `list` é puramente leitura e podia ser entregue sozinho, antes do `add`. Foi o que aconteceu.

### As duas decisões que ficavam em aberto

Esta ADR deixava duas perguntas sem resposta, e as duas foram fechadas na [SRS — ignore](../srs/ignore.md):

- **Sem flag de destino, grava em `<raiz>/.gitignore`** — o padrão do comando é a convenção de time, que é o que o próprio git recomenda.
- **`--global` recusa quando o `core.excludesFile` não está configurado**, salvo com `--force`, que então configura **e** cria o arquivo na mesma chamada. Escrever um arquivo que o git não vai ler seria pior que não escrever nada.

## O que a entrega do `list` decidiu além desta ADR

- **Cabeçalho na saída de texto.** As três colunas saem sob `CAMINHO`, `ORIGEM` e `PADRÃO`. Sem rótulo a saída é ambígua de verdade: `node_modules/` aparece na coluna do caminho **e** na do padrão, e não há como o leitor saber qual é qual. `ORIGEM` e não `REGRA` porque o padrão *é* a regra — o que a coluna do meio carrega é de onde a regra veio.
- **A ordem das colunas no texto é outra que a do CSV.** No texto o caminho vem primeiro, porque é o que se procura. O `Entry` guarda a ordem da pipeline — origem, linha, padrão, caminho — e é ela que sai no CSV.
- **`Entry.Directory`.** A barra final é o git reportando o que o `--directory` colapsou, então quem a interpreta é o domínio. A apresentação lê um booleano em vez de conhecer a convenção de saída do `ls-files`.
- **Registro truncado é descartado.** O parse anda de quatro em quatro campos e ignora a cauda incompleta, sem desalinhar o resto.

### Limitação conhecida

O `CommandRunner.run` faz `strings.TrimSpace` na saída crua. Com `-z` o NUL final protege a cauda, mas um caminho cujo **primeiro** caractere seja espaço perderia esse espaço. É a mesma família da armadilha 1. Consertar exige um caminho de saída crua no port; não foi feito.

## Relacionadas

- **[SRS — ignore](../srs/ignore.md)** — os requisitos do comando, e onde as duas decisões em aberto foram fechadas.
- **[ADR-004](004-configuracao-e-formatos.md)** — `--format`, `--separator` e `ignored.separator`.
- **[SRS — equivalência por conteúdo](../srs/equivalencia.md)** — evitou estender o port usando `commit-tree`; a dívida ficou para cá.
