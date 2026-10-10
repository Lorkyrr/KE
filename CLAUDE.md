# CLAUDE.md

KE (KEngine 2.0): biblioteca Go de quadros, operadores e receitas, mais o firewall que recebe e entrega os dados. Visão geral e referência das funções em [README.md](README.md). Decisões em [docs/adr/](docs/adr/).

Este é um repositório **Go, só stdlib**. As regras de Kotlin/Micronaut/React do `CLAUDE.md` global não se aplicam aqui.

> **Projeto em pausa desde 2026-10-10.** Não implemente funcionalidade nova. A intenção é seguir como biblioteca de transpilação para Python (classes Python compiladas para Go, dentro do interpretador Python, para dar multi-thread leve ao Python); o desenho ainda não existe. Contexto e perguntas em aberto: [ADR 0003](docs/adr/0003-pausa-e-direcao-transpilador-python.md).

## Idioma

Português do Brasil, com acentos, em código, comentários, mensagens de erro, testes e commits. Identificadores de domínio ficam em português (`Quadro`, `Faixa`, `Receita`, `Juncao`); erros são `ErrXxx` e dizem o pacote (`"quadro: tag repetida"`).

## Comandos

Go 1.27.1 fica em `~/sdk/go1.27.1/bin`; `staticcheck` e `govulncheck` em `~/go/bin`. Se `go` não estiver no `PATH`, exporte os dois antes.

```bash
test -z "$(gofmt -l .)" && go vet ./... && GOOS=windows go vet ./...
go test -race ./...
go test -run x -bench . -benchmem ./quadro/
go test -run x -fuzz FuzzIndexar -fuzztime 30s ./quadro/
```

Antes de dizer que algo está pronto, rode os dois primeiros e cole a saída. `gofmt -l` só passa com saída vazia.

## Regras que os testes impõem (não contorne)

1. **Zero-cópia.** O dado entra no quadro uma vez. Depois só passam `*Quadro` e `Faixa`. Nada de `string(campo)`, `append` para copiar campo, nem buffer intermediário no caminho fio → contrato → receita → fio. `TestCaminhoNaoAloca` e `TestTransporteNaoAloca` exigem 0 alocações (o segundo fica fora do `-race`, que faz o `sync.Pool` descartar itens); um `AllocsPerRun` novo vale para qualquer caminho novo.
2. **ISA.** Fora de `operadores/`, ninguém escreve `func(*quadro.Quadro) error`. Processo é `Seq` de operadores. Falta instrução: ela entra em `operadores/`, e só quando uma receita real pedir.
3. **Fronteiras de import.** Só `firewall/router/externo` importa `net`. `quadro`, `operadores` e `pipeline` importam só stdlib e a si mesmos. `sanitizar` não importa o KE; `pipeline` não importa `firewall`; `entrada` não importa `pipeline` nem `operadores`. Mudou o grafo? Rode `arquitetura_test.go`.
4. **Sem `unsafe` e sem assembly** (spec v2 §7.5). Slices já dão endereço e posição com checagem de limite.

## Invariantes do quadro

- Capacidade fixa de 64 KiB. Nunca realoca: sem espaço, a operação **recusa** (`ErrCheio`).
- `Campo` devolve `buf[ini:fim:fim]`. Mantenha a capacidade cortada.
- `Indexar` e `Promover` são atômicos: erro não deixa o índice pela metade.
- `Zerar` sobrescreve os bytes (dado pessoal não fica em memória reaproveitada). Não troque por reslice.
- Posse vai com o ponteiro. Quem encaminhou, entregou ou devolveu não toca mais no quadro. Escrita em quadro solto recusa com `ErrSolto`.
- Entrada de até 32 KiB (`MaxEntrada`), para a resposta sempre caber no mesmo buffer.

## Convenções de código

- Dinheiro é `int64` em centavos. Toda conta confere estouro; nunca deixe dar a volta em silêncio.
- Divisão inexata nunca some: `Dividir` recusa (`ErrInexato`); quem precisa de quociente usa `Arredondar` com o modo explícito.
- Registrador ou modo inválido dá `panic` **na declaração** do operador, nunca na execução. É o único `panic` aceito; entrada externa (fio, arquivo, usuário) devolve erro.
- Erro embrulhado com `%w`, para `errors.Is`/`errors.As` funcionarem. Nunca atribua erro a `_` fora de teste. `Recusa` e mensagens de erro **não carregam o valor do campo** (pode ser dado pessoal).
- Comentário explica o porquê, não o quê. Pacote exportado tem comentário de documentação.
- Teste novo: nome em português dizendo o comportamento (`TestJuncaoCheiaRecusa`), `-race` limpo, e fuzz para qualquer parser novo.

## Em aberto: não implemente sem aval

O [ADR 0002](docs/adr/0002-transmitir-receitas-e-fluxo-entre-pipelines.md) lista pendências (quadro privado por execução, corrida no `Ligar`, `Juncao` executando na goroutine de quem entrega, `MaxOrigens = 4`, onde mora o adapter, e outras). `Transmitir` do quadro inteiro existe no código; a transmissão **por campo** está decidida no papel e **não existe**. Pergunta sem resposta escrita continua sem resposta: pare e liste a pergunta. A primeira coisa a escrever, para a pendência 1, é o teste de duas receitas concorrentes sob `-race`, antes de qualquer correção.

## Processo (diretrizes e agentes)

- [Diretrizes-de-Codigo.md](Diretrizes-de-Codigo.md) (v2.2.1) manda. Ordem de prioridade em conflito: Segurança, Confiabilidade, Simplicidade, Modularidade, Velocidade, Autonomia. Nada se perde em silêncio. Desvio exige comentário ou ADR.
- Nível do projeto: veja o bloco `diretrizes` do README. Os comandos que provam cada item estão em [Conformidade-Diretrizes.md](Conformidade-Diretrizes.md) (seção Go).
- [agentes.md](agentes.md) diz quem faz o quê, com qual modelo e esforço. Resumo: spec antes de código; o contrato é o spec e a mensagem só aponta o caminho; mecânico vai a subagente Haiku em `low`; invariante (estado, posse, dinheiro) é Opus. Devolva só: arquivos mexidos, comando de teste com a saída, o que ficou de fora.
- Fechamento de feature: uma linha em `.planning/ROADMAP.md` (agentes.md §7). Não reescreva README, ADR nem changelog por conta própria; é outro pedido.
- ADR novo: `git fetch` e confira a numeração antes (sessões paralelas colidem). Modelo em `docs/adr/0000-modelo.md` quando existir.
- Commits pequenos, Conventional Commits em português (`feat(quadro): ...`, `docs: ...`), um assunto por commit.
