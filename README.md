# KE — KEngine 2.0

Motor de integração multiplataforma. O cerne é o **firewall**, com o router,
e os **operadores**. Desenho:
[`KEngine/.planning/specs/kengine-2-0-arquitetura.md`](https://github.com/Lorkyrr/KEngine)
e [ADR 0001](docs/adr/0001-quadro-faixas-e-operadores-como-biblioteca.md).

```text
quadro/                biblioteca: buffer por comando, faixas (posições), registradores e Ligar (fan-in)
operadores/            biblioteca: as instruções (Converter, Somar, Subtrair, Multiplicar, Dividir,
                       Arredondar, Escrever) e Seq
pipeline/              biblioteca: Receita (entradas exigidas + operadores) e Juncao (entrada sincronizada)
firewall/
├── sanitizar/         folhas puras sobre []byte
├── entrada/           contrato declarado por tag; sem declaração, derruba
└── router/
    ├── externo/       transporte; o único que importa net
    └── interno/       encaminha o endereço do quadro, nunca o conteúdo
```

## Duas regras, impostas por teste

- **Zero-cópia.** O dado entra no quadro uma vez, direto do fio. Daí em diante
  passam só `*Quadro` e `Faixa`, e a resposta sai do mesmo buffer.
  `TestCaminhoNaoAloca` cobra 0 alocações.
- **ISA.** Processo é uma sequência de operadores da biblioteca. Fora de
  `operadores/`, ninguém escreve `func(*quadro.Quadro) error`
  (`TestSoAOperadoresDefineInstrucoes`).

## Usar a biblioteca em outro projeto

```go
import (
    "github.com/lorkyrr/ke/operadores"
    "github.com/lorkyrr/ke/quadro"
)

total := operadores.Seq(
    operadores.Converter(operadores.R0, tagPreco),
    operadores.Converter(operadores.R1, tagQuantidade),
    operadores.Multiplicar(operadores.R2, operadores.R0, operadores.R1),
    operadores.Escrever(tagTotal, operadores.R2),
)
```

`quadro` e `operadores` dependem só da stdlib (`TestBibliotecaSoDependeDaStdlib`).

## Comandos

```bash
test -z "$(gofmt -l .)" && go vet ./... && GOOS=windows go vet ./...
go test -race ./...
go test -run x -bench . -benchmem ./quadro/
go test -run x -fuzz FuzzIndexar -fuzztime 30s ./quadro/
```
