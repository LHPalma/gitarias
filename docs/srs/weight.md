---
titulo: SRS — weight
data: 2026-08-15
status: entregue
comando: gtr weight
apelido: roadie
pacotes:
  - cmd/weight.go
  - cmd/weight_table.go
  - internal/weight
  - internal/ui
commits:
  - 632d059
fonte_externa: nenhuma
---

# SRS — weight

- **Data:** 2026-08-15
- **Feature:** `cmd/weight` + `internal/weight`
- **Status:** **entregue** — commit `632d059` na `main`
- **Fonte externa:** nenhuma. Plumbing de leitura, sem rede

---

## 1. Introdução

### 1.1 Propósito

Especificar o `gtr weight`, que responde **o que está pesando no histórico do repositório**.

**A premissa que quase ninguém conhece:** apagar um arquivo não o remove do repositório. O commit em que ele existia continua no histórico, o objeto segue **alcançável**, e **todo clone continua baixando aquele blob — para sempre**, inclusive quem entrou anos depois e nunca vai ver o arquivo.

### 1.2 O mecanismo, medido

```console
=== arquivo removido num commit à frente, com git gc --prune=now:
    size-pack: 2.86 MiB          ← o peso continua lá
    na árvore?        0 ocorrências
    alcançável?       1 referência

=== tirando do histórico os commits que o continham:
    size-pack: 1.89 KiB          ← 2,86 MB → 1,89 KB
    alcançável?       0 referências
```

**O critério do git não é "está na árvore", é "é alcançável".** O git guarda fotografias: o commit onde o arquivo existia tem uma árvore que aponta para aquele blob, e commit é imutável. Remover no commit seguinte só significa que a árvore **do commit novo** não o menciona.

**Por isso o `git gc` não resolve.** O `gc` recolhe *lixo* — objeto que ninguém alcança. O blob não é lixo, é história.

### 1.3 A lacuna que o comando preenche

| Ferramenta | Responde | Não responde |
|---|---|---|
| `du` da árvore | o que você vê | o que o clone baixa |
| `git count-objects -vH` | **quanto** pesa | **quem** pesa |
| **`gtr weight`** | quem pesa, **e se ainda está na árvore** | — |

### 1.4 O nome

**`weight`** — o peso, não o tamanho. Tamanho é do arquivo; peso é o que ele custa a quem clona.

**`roadie` é apelido**: quem carrega o equipamento pesado da banda. Terceiro do elenco, ao lado do `soundcheck` e do `rewind`. Fora da ajuda da raiz.

---

## 2. Descrição geral

### 2.1 Posição na arquitetura

```text
internal/weight/             domínio do peso — não imprime nada
├── runner.go                a interface maior, com RunWithInput
├── path.go                  Path: caminho, bytes, versões, InTree
├── report.go                Report: total e caminhos
└── repo.go                  Heaviest, sizes, tracked, total

internal/ui/bytes.go         DescribeBytes: o tamanho para gente
internal/ui/residence.go     DescribeResidence: na árvore / só no histórico
cmd/weight.go                o comando, o --limit e a manchete
cmd/weight_table.go          a tabela, nos quatro formatos
```

### 2.2 O cano

```console
git rev-list --objects --all
  → todo objeto alcançável de qualquer ref, com o caminho de cada um

  ↓ pela entrada padrão

git cat-file --batch-check='%(objecttype) %(objectname) %(objectsize:disk) %(rest)'
  → tipo e tamanho EM DISCO de cada um

git ls-tree -r -z --name-only HEAD
  → o que ainda está na árvore

git count-objects -v
  → o total: soltos + empacotados
```

Os dois primeiros foram feitos para conversar por `stdin`, e é assim que o `RunWithInput` os liga — sem shell.

---

## 3. Requisitos funcionais

### RF-01 — Pesar o histórico por caminho

Soma o tamanho **em disco** de todas as versões de cada caminho e ordena do maior para o menor. Empate desempata por nome, para que a saída não mude entre execuções.

### RF-02 — Marcar o que já saiu da árvore

Cada caminho sai como `na árvore` ou `só no histórico`. **É esse o achado do comando:**

```console
árvore de trabalho: 8.0K

Um clone deste repositório baixa 2.9 MB.

  2.9 MB  1 versão   build.zip  só no histórico
  34 B    2 versões  app.go     na árvore
```

### RF-03 — Limitar e emitir

`--limit N`, padrão 10, com `0` trazendo todos. Os quatro formatos da **ADR-004**, e a manchete com o total **fora de todo formato contratual**.

---

## 4. Regras de negócio

| ID | Regra |
|---|---|
| **RN-01** | **O tamanho medido é o em disco, não o lógico.** `objectsize:disk` é o que um clone paga: texto que muda pouco vira poucos bytes no pack, enquanto um zip **não comprime nem faz delta** com nada. O tamanho lógico mentiria nos dois sentidos. |
| **RN-02** | **Caminho cru dos dois lados do cruzamento.** O `rev-list --objects` entrega caminhos crus e o `ls-tree --name-only` **cita e escapa em C**. O `-z` é obrigatório — ver §5.1. |
| **RN-03** | **Diagnostica e não conserta.** Expurgar um blob é reescrever histórico: todo SHA dali para frente muda, todo clone existente quebra, todo PR aberto quebra. Isso é conversa de **ADR-008**, e o comando existe para dizer **se vale a pena** tê-la. |

### 4.1 Regras herdadas

- **A leitura vem de formato contratual** — `RN-05` da [SRS — branches](branches.md). `rev-list`, `cat-file --batch-check`, `ls-tree -z`, `count-objects -v` — nada de `-vH`, cuja unidade é texto para gente.
- **A manchete não entra no `stdout` contratual** — `RN-09` da [SRS — branches](branches.md). No `csv`, `tsv` e `json` ela vai para o `stderr`; no `json` o total já está no envelope, como `total_bytes`. Ver §5.2.
- **Nenhuma linha termina em espaço** — `RN-10` da [SRS — branches](branches.md), pelo `trimmingWriter`.
- **O domínio não formata** — `RN-11` da [SRS — branches](branches.md). `Path.Bytes` é `int64`; `2.9 MB` é `ui.DescribeBytes`.

**Dois vocabulários, como no resto do projeto:** a tela mostra `2.9 MB`, o `csv` e o `json` mostram `3000000`. Quem lê é planilha ou script.

---

## 5. Achados de implementação

### 5.1 A citação em C do `ls-tree`, e o bug que ela produziria

Medido antes de escrever, e é a armadilha central desta feature:

```console
$ git rev-list --objects --all
975fbec... com "aspas".txt          ← cru
587be6b... revisão.txt              ← cru

$ git ls-tree -r HEAD --name-only
"com \"aspas\".txt"                  ← citado e escapado
"revis\303\243o.txt"                 ← citado e escapado

$ git ls-tree -r -z HEAD --name-only
com "aspas".txt                     ← cru de novo
revisão.txt
```

Cruzar as duas listas sem o `-z` marcaria **como "só no histórico" todo arquivo com acento, aspas ou caractere especial** — errado, e silenciosamente. O `core.quotePath` não afeta o `rev-list`, então nem havia como alinhar pelo outro lado.

**É a mesma família do motivo de lock escapado do `worktrees`.** Segunda ocorrência da mesma armadilha, e a lição tem duas para sustentá-la: **quando dois comandos do git alimentam o mesmo cruzamento, conferir se ambos escapam igual.**

Conferido no binário com nomes contendo acento, aspas e espaço — os três classificados certo.

### 5.2 O JSON saía inválido, e o teste pegou

A manchete usava `chosen.format.Delimited()` para decidir entre `stdout` e `stderr`. **O `Delimited()` cobre `csv` e `tsv`, mas não `json`** — então o documento saía assim:

```text
Um clone deste repositório baixa 2.9 MB.

{
  "total_bytes": 3000320,
  ...
```

JSON inválido, quebrando a leitura contratual na cara. A condição certa não era "é delimitado", era **"é o formato de tela"**. O teste agora percorre os três formatos contratuais em vez de só o `csv`.

### 5.3 Um teste achou código morto

A cobertura apontou um statement inalcançável: o `tracked` nunca devolvia erro — repositório sem `HEAD` devolve mapa vazio de propósito —, então o `if err != nil` depois da chamada era decorativo. **O erro saiu da assinatura**, em vez de manter uma checagem que fingia poder acontecer.

### 5.4 O total soma solto e empacotado

`count-objects -v` informa `size` e `size-pack` **separados, os dois em KiB**. Um repositório recém-commitado tem **tudo solto e nada no pack** — usar só o `size-pack` reportaria zero justamente no caso mais comum de quem está investigando.

---

## 6. Testes

**100% de statements** em `cmd`, `internal/weight` e `internal/ui`.

| # | Cenário | Esperado |
|---|---|---|
| 1 | Duas versões do mesmo caminho | Somadas, com a contagem — `RF-01` |
| 2 | Arquivo removido da árvore | `só no histórico` — `RF-02` |
| 3 | Empate de peso | Desempate por nome: a ordem não pode mudar entre rodadas |
| 4 | `--limit` menor e maior que o histórico | Corta os menores, nunca os maiores — `RF-03` |
| 5 | `commit` e `tree` na saída do `cat-file` | Ignorados: não têm caminho |
| 6 | Caminho com espaço | Preservado inteiro |
| 7 | Repositório sem objetos, e sem `HEAD` | Sem erro; sem `HEAD`, nada está na árvore |
| 8 | Falha em cada um dos três comandos | Erro propagado — em tabela |
| 9 | A lista vai pelo `stdin` | É assim que os dois comandos conversam |
| 10 | Manchete em `csv`, `tsv` e `json` | Sempre no `stderr` — §5.2 |
| 11 | `DescribeBytes` de 0 a 1 PB | Sobe de unidade na hora certa |
| 12 | `roadie` e a ajuda da raiz | Mesma saída; apelido não anunciado |

**Conferido por mutação:** invertendo a marcação da árvore, três testes caem; tirando o `-z` do `ls-tree`, três testes caem.

---

## 7. Rastreabilidade

| Commit | Entrega |
|---|---|
| `632d059` | O domínio, os rótulos de tamanho e residência, o comando com `--limit` e o `roadie` — `RF-01` a `RF-03`, `RN-01` a `RN-03` |

Verificado pelo `gtr commits check` antes de ir para a `main`.

---

## 8. Follow-ups conhecidos

- **Não distingue quem ainda alcança o blob.** Um arquivo pode estar fora do `HEAD` e continuar numa tag ou noutra branch — o comando diz `só no histórico` nos dois casos. Mostrar **de onde** ele ainda é alcançável (`git tag --contains`, `branch --contains`) diria se o expurgo é viável.
- **Não agrupa por diretório.** `assets/` inteiro pesando 40 MB em mil arquivos pequenos não aparece; mil linhas, sim.
- **Não olha o que está fora do `--all`.** Objeto alcançável só pelo reflog não é contado, e ele também ocupa disco — mas não é baixado por um clone, então fica fora do propósito.
- **O expurgo é assunto de outra feature, se um dia.** `filter-repo` é destrutivo do tipo mais grave que existe: reescreve o histórico inteiro. Cairia sob as três guardas da **ADR-008**, e não é óbvio que embrulhá-lo acrescente alguma.
