# KE

Motor de integração multiplataforma (KEngine 2.0): uma biblioteca de quadros, operadores e receitas, e o firewall que recebe e entrega os dados.

## Nível do projeto

**Nível 2**: biblioteca embutida por outros projetos. O KE não disca, não escuta, não autentica e não persiste nada (`Receber` e `Enviar` recebem `io.Reader` e `io.Writer` prontos). TLS, autenticação, timeout e retry são de quem abre a conexão; a concorrência é coberta por `-race` em todo teste. Se o KE passar a abrir conexões ou guardar dados, sobe para o nível 3. Exigências de cada nível nas [Diretrizes de Código](Diretrizes-de-Codigo.md).

```diretrizes
nome: KE
nivel: 2
linguagens: go
resumo: Motor de integração multiplataforma (KEngine 2.0): uma biblioteca de quadros, operadores e receitas, e o firewall que recebe e entrega os dados.
lint: go vet ./... && GOOS=windows go vet ./...
testar: go test -race ./...
ecossistemas: gomod
```

## Status

Em desenvolvimento. Funcionam hoje: o quadro com faixas, registradores, fan-in (`Ligar`), `Promover`, posse e rastro (`Soltar`/`Acoplar`); os operadores aritméticos e de conversão; `Receita` e `Juncao`; o firewall (sanitizar, contrato por tag, transporte e rota). O caminho fio → quadro → contrato → rota → receita → fio é coberto de ponta a ponta por `TestPontaAPonta`.

**Ainda não existe** o operador `Transmitir`, que passa um campo de uma pipeline para outra. Está decidido no papel ([ADR 0002](docs/adr/0002-transmitir-receitas-e-fluxo-entre-pipelines.md)), com oito pendências. Duas afetam quem usa a biblioteca hoje:

- `Reg` mora no quadro: duas receitas rodando ao mesmo tempo **sobre o mesmo quadro** sobrescrevem os registradores uma da outra. Use um quadro por execução.
- `Campo` de um quadro ligado (`Ligar`) só é seguro enquanto a origem não é mais tocada.

Hooks (`.githooks/`), CI e `CHANGELOG.md` das diretrizes ainda não foram instalados neste repositório.

## Instalação

Requer Go 1.27.1 (`go.mod`). Só a stdlib; nenhuma dependência externa.

```bash
go get github.com/lorkyrr/ke
```

`quadro`, `operadores` e `pipeline` são a biblioteca: quem importa só `operadores` compila só `operadores` e `quadro` (`TestBibliotecaSoDependeDaStdlib`). O `firewall/` é o que o KE usa para receber e entregar; um consumidor pode usá-lo ou escrever o próprio transporte.

## Uso

### Duas regras, impostas por teste

- **Zero-cópia.** O dado entra no quadro uma vez, direto do fio. Daí em diante passam só `*Quadro` e `Faixa`, e a resposta sai do mesmo buffer. `TestCaminhoNaoAloca` (contrato → receita) e `TestTransporteNaoAloca` (`Receber` e `Enviar`) cobram 0 alocações.
- **ISA.** Processo é uma sequência de operadores da biblioteca. Fora de `operadores/`, ninguém escreve `func(*quadro.Quadro) error` (`TestSoAOperadoresDefineInstrucoes`). Falta uma instrução? Ela entra na biblioteca.

### Exemplo: total de um item

Uma receita montada como um projeto consumidor monta, só com operadores:

```go
const (
    tagPreco quadro.Tag = iota + 1
    tagQuantidade
    tagTotal
)

totalItem := pipeline.Declarar("total-item",
    []quadro.Tag{tagPreco, tagQuantidade}, // entradas exigidas
    operadores.Converter(operadores.R0, tagPreco),
    operadores.Converter(operadores.R1, tagQuantidade),
    operadores.Multiplicar(operadores.R2, operadores.R0, operadores.R1),
    operadores.Escrever(tagTotal, operadores.R2),
)

q := quadro.Pegar()
defer quadro.Devolver(q)

externo.Receber(conn, q)          // fio → buffer do quadro, uma única vez
entrada.Verificar(q, &contrato)   // indexa e aplica o contrato por tag
totalItem.Executar(q)             // roda a sequência; erro diz a etapa
externo.Enviar(conn, q)           // a saída sai do mesmo buffer
```

(Os `error` foram omitidos por brevidade; trate cada um. O teste completo está em `ponta_a_ponta_test.go`.)

### Formato no fio

Cada mensagem é `[tamanho u32][campos]`. Cada campo é TLV, em big-endian:

```text
[tag u16][tamanho u16][valor]
```

Tags de 0 a 63. Entrada de no máximo 32 KiB (metade do quadro, para a resposta sempre caber).

### Referência: `quadro`

O buffer de um comando e o índice de onde cada campo está. Capacidade fixa de 64 KiB: nunca cresce, então nada se move por trás de quem guarda uma `Faixa`.

| Símbolo | O que faz |
| --- | --- |
| `Tag` | Identifica um campo (`uint16`, valores de 0 a `MaxTags-1`) |
| `Faixa{Ini, Fim}` | Endereço de um campo dentro do buffer. `Tamanho()` devolve os bytes |
| `Novo()` | Quadro fora do pool (testes, uso isolado) |
| `Pegar()` | Quadro vazio do pool |
| `Devolver(q)` | Zera e devolve ao pool, junto com as origens ligadas. Depois disso, não toque em `q` nem em fatias obtidas dele |
| `q.Zerar()` | Sobrescreve os bytes usados (dado pessoal não fica em memória reaproveitada) e limpa índice, registradores e rastro |
| `q.Reservar(n)` | Marca `n` bytes de entrada e devolve a área para o transporte preencher direto (`io.ReadFull`). `ErrOcupado` se já tem conteúdo; `ErrCheio` se não cabe |
| `q.Indexar()` | Percorre a entrada uma vez e marca a faixa de cada tag. Atômico: se algo for inválido, o índice não muda |
| `q.Presente(t)` | A tag é visível (própria ou de uma origem ligada)? |
| `q.Faixa(t)` | Posição do campo no buffer **próprio**. Campo de origem não tem faixa aqui |
| `q.Campo(t)` | Janela sobre o campo, **sem copiar**. A capacidade é cortada no fim do campo: um `append` de quem recebe não escreve sobre o vizinho |
| `q.Saida()` | O que foi anexado depois da entrada, pronto para o transporte |
| `q.Anexar(t, valor)` | Escreve um campo na saída, no mesmo buffer. Sem capacidade, recusa (`ErrCheio`) em vez de realocar |
| `q.AnexarInt(t, v)` | Escreve `v` em decimal direto no buffer, sem string intermediária |
| `AnexarTLV(dst, t, valor)` | Codifica um campo em `dst`. É o lado de quem monta a mensagem (cliente, testes) |
| `q.Ligar(o)` | Fan-in: `q` enxerga os campos de `o` sem copiá-los. A posse de `o` passa para `q`. Máximo `MaxOrigens` (4); tag já visível recusa (`ErrRepetida`) |
| `q.Promover()` | Torna a saída já escrita em entrada indexada, no mesmo buffer: é como a saída de uma pipeline vira a entrada da próxima sem ser copiada |
| `q.Soltar(pipeline)` | Desencaixa o quadro da pipeline: quem soltou não escreve mais nele (`ErrSolto`) |
| `q.Acoplar(pipeline)` | Encaixa o quadro numa pipeline. Só vale em quadro solto (`ErrAcoplado`) |
| `q.Solto()` | O quadro está em trânsito, sem dono? |
| `q.Marcas(dst)` | Rastro dos últimos `MaxMarcas` (8) encaixes e desencaixes, do mais antigo ao mais novo. Sem alocar se `dst` tiver capacidade |
| `q.Reg` | Banco de `MaxRegs` (8) registradores `int64`, onde os operadores guardam valores intermediários |

Constantes: `MaxTags` 64, `MaxRegs` 8, `MaxOrigens` 4, `Capacidade` 64 KiB, `MaxMarcas` 8, `CabecalhoCampo` 4, `MaxValor` 65535.

Erros: `ErrCorte` (campo passa do fim da entrada), `ErrTag` (tag fora do intervalo), `ErrRepetida`, `ErrCheio`, `ErrOcupado`, `ErrOrigem`, `ErrSolto`, `ErrAcoplado`.

### Referência: `operadores`

As instruções. Um `Op` é `func(*quadro.Quadro) error`; cada operador lê campos pela posição no buffer e guarda valores nos registradores `R0` a `R7`. Registrador ou modo inexistente **dá `panic` ao montar** o programa, nunca no meio de um comando.

| Operador | O que faz |
| --- | --- |
| `Seq(ops...)` | Compõe operadores em ordem; para no primeiro erro e diz a etapa (`etapa 2: ...`). É a única forma de montar um processo |
| `Converter(dst, tag)` | Lê os dígitos ASCII do campo direto da faixa para o registrador. Só dígitos 0-9, sem sinal: `ErrAusente`, `ErrFormato` ou `ErrEstouro` |
| `Escrever(tag, src)` | Anexa o registrador à saída, em decimal |
| `Somar(dst, a, b)` | `dst = a + b` |
| `Subtrair(dst, a, b)` | `dst = a - b` |
| `Multiplicar(dst, a, b)` | `dst = a * b` |
| `Dividir(dst, a, b)` | `dst = a / b`, **só divisão exata**: o resto nunca some em silêncio (`ErrInexato`, `ErrDivisaoPorZero`) |
| `Arredondar(dst, a, b, modo)` | `dst = a / b` arredondado pelo `Modo`. Com divisão exata, dá o mesmo que `Dividir` |

Modos de `Arredondar`: `Truncar` (em direção a zero), `Piso` (a -∞), `Teto` (a +∞), `MeioAcima` (ao mais próximo; metade vai para longe de zero), `MeioPar` (ao mais próximo; metade vai para o par, bancário).

Dinheiro é inteiro (centavos). Toda conta confere estouro e devolve `ErrEstouro` em vez de dar a volta em silêncio.

### Referência: `pipeline`

| Símbolo | O que faz |
| --- | --- |
| `Declarar(nome, entradas, etapas...)` | Monta uma `Receita`: as tags exigidas e a sequência de operadores |
| `r.Executar(q)` | Roda a receita sobre o quadro. Faltando entrada, devolve `ErrAguardando` (confira com `errors.Is`); não é falha, é o estado de espera |
| `NovaJuncao(r, capacidade)` | Entrada sincronizada: junta os pedaços de uma mesma entidade vindos de pipelines anteriores. A tabela tem tamanho fixo |
| `j.Entregar(chave, o)` | Junta o quadro `o` à entidade `chave`. Sem todas as entradas, devolve `ErrAguardando`. Quando a última chega, executa a receita e devolve o quadro da entidade (com a `Saida`); quem recebe chama `quadro.Devolver` nele, **mesmo com erro** |
| `j.Pendentes()` | Quantas entidades esperam entradas |

Posse em `Entregar`: em sucesso ou espera, `o` passa à junção. Em qualquer outro erro (`ErrRepetida`, tabela cheia com `ErrCheio`), `o` continua de quem entregou e a entidade segue como estava. Tabela cheia é contrapressão: a recusa chega a quem entrega.

A receita roda na goroutine de quem entrega a última entrada (pendência 3 do ADR 0002).

### Referência: `firewall`

A única entrada e a única saída de dados do KE.

| Pacote | Símbolo | O que faz |
| --- | --- | --- |
| `sanitizar` | `Digitos(v)` | Aceita só 0-9 e pelo menos um dígito. Folha pura sobre `[]byte`: sem estado, sem string, sem cópia |
| `entrada` | `Contrato` | Array indexado pela tag; consultar é acessar uma posição |
| `entrada` | `Campo{Max, Obrigatorio, Sanitizar}` | Declaração de uma tag. `Max == 0` quer dizer **não declarada** |
| `entrada` | `Verificar(q, c)` | Indexa o quadro e aplica o contrato, barato primeiro: declaração, presença, tamanho, formato. Tag sem declaração **derruba** o comando |
| `entrada` | `*Recusa{Tag, Motivo, Err}` | Diz qual campo derrubou o comando e por quê. Nunca carrega o valor (pode ser dado pessoal). Use `errors.As` |
| `router/externo` | `Receber(r, q)` | Lê uma mensagem do fio direto para dentro de `q`. O limite (`MaxEntrada`, 32 KiB) é conferido antes de ler o corpo (`ErrGrande`) |
| `router/externo` | `Enviar(w, q)` | Manda a `Saida` do quadro. Usa `net.Buffers` (writev): cabeçalho e corpo saem sem serem concatenados |
| `router/interno` | `Nova(capacidade)` | Cria uma `Rota` com limite |
| `router/interno` | `r.Encaminhar(q)` | Entrega o quadro à rota ou recusa na hora (`ErrRotaCheia`, fail-fast). Pela rota passa só o endereço (8 bytes), nunca o conteúdo |
| `router/interno` | `r.Recebidos()` / `r.Fechar()` | Lado do consumidor / encerramento (só quem produz fecha) |

Posse: quem encaminha entrega o quadro junto com o ponteiro e não toca mais nele. Quem recebe passa a ser o dono e o devolve ao pool no fim. Em caso de recusa, a posse continua com quem chamou.

## Desenvolvimento

```bash
test -z "$(gofmt -l .)" && go vet ./... && GOOS=windows go vet ./...
go test -race ./...
go test -run x -bench . -benchmem ./quadro/
go test -run x -fuzz FuzzIndexar -fuzztime 30s ./quadro/
```

Os comandos `lint` e `testar` do bloco `diretrizes` acima são os que o `estrutura.sh` executa. O padrão da linguagem (Go) também pede `staticcheck ./...` e `govulncheck ./...`; o primeiro não está no `lint` declarado.

O que `arquitetura_test.go` cobra, em qualquer mudança:

| Regra | Teste |
| --- | --- |
| Só `firewall/router/externo` importa `net`, `net/http` ou `database/sql` | `TestSoORouterExternoImportaRede` |
| `quadro`, `operadores` e `pipeline` só importam stdlib e a si mesmos | `TestBibliotecaSoDependeDaStdlib` |
| `sanitizar` não importa o KE; `pipeline` não importa `firewall`; `entrada` não importa `pipeline` nem `operadores` | `TestDirecaoDasCamadas` |
| Nenhuma `func(*quadro.Quadro) error` fora de `operadores/` | `TestSoAOperadoresDefineInstrucoes` |

Para quem trabalha com agentes (Claude Code, Grok), as regras do repositório estão em [CLAUDE.md](CLAUDE.md), e quem faz o quê, com qual modelo e esforço, em [agentes.md](agentes.md).

## Arquitetura

| Módulo | Responsabilidade | Depende de |
| --- | --- | --- |
| `quadro` | Buffer por comando, faixas (posições), registradores, `Ligar` (fan-in), `Promover`, posse e rastro | stdlib |
| `operadores` | As instruções e `Seq` | `quadro` |
| `pipeline` | `Receita` (entradas exigidas + operadores) e `Juncao` (entrada sincronizada) | `operadores`, `quadro` |
| `firewall/sanitizar` | Folhas puras sobre `[]byte` | stdlib |
| `firewall/entrada` | Contrato declarado por tag; sem declaração, derruba | `quadro` |
| `firewall/router/externo` | Transporte; o único que importa `net` | `quadro` |
| `firewall/router/interno` | Encaminha o endereço do quadro, nunca o conteúdo | `quadro` |

```text
quadro/                biblioteca
operadores/            biblioteca
pipeline/              biblioteca
firewall/
├── sanitizar/
├── entrada/
└── router/
    ├── externo/
    └── interno/
```

## Decisões

Decisões de arquitetura e desvios das diretrizes ficam em `docs/adr/`.

- [ADR 0001](docs/adr/0001-quadro-faixas-e-operadores-como-biblioteca.md): quadro com faixas (zero-cópia) e operadores como biblioteca.
- [ADR 0002](docs/adr/0002-transmitir-receitas-e-fluxo-entre-pipelines.md): `Transmitir`, receitas como mini-threads e fluxo entre pipelines. Lista as pendências de desenho.

O desenho completo está no spec do KEngine: [`KEngine/.planning/specs/kengine-2-0-arquitetura.md`](https://github.com/Lorkyrr/KEngine).

## Licença

Não definida.
