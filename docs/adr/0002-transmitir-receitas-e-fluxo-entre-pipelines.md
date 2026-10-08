# 0002. Transmitir, receitas como mini-threads e fluxo entre pipelines

- **Status:** parcial. A base está implementada; o desenho do fluxo por campo
  está decidido no papel, com pendências listadas abaixo.
- **Data:** 2026-10-08

## Contexto

O ADR 0001 fixou o quadro (buffer zero-cópia) e os operadores como biblioteca.
Faltava dizer como dados passam de uma pipeline para outra. O pedido: cada
entrada é uma struct numerada, com vários tipos (tags) processados por funções
separadas, que não terminam juntas. Um tipo que termina deve seguir adiante
sem esperar os outros.

## Decisões implementadas

1. **Três pacotes de biblioteca em um `go.mod`:** `quadro`, `operadores` e
   `pipeline` (que saiu de `nucleo/`). Seguem separados, e não num pacote só,
   porque a fronteira de pacote impõe o encapsulamento do buffer, a regra ISA
   e a dependência só de stdlib (`arquitetura_test.go`).
2. **Aritmética:** `Subtrair`; `Dividir` só aceita divisão exata
   (`ErrInexato`); `Arredondar(dst, a, b, modo)` é operador à parte, com os
   modos `Truncar`, `Piso`, `Teto`, `MeioAcima` e `MeioPar`.
3. **`quadro.Ligar`:** um quadro enxerga campos de outros sem copiar (fan-in).
   `quadro.Promover` torna a saída uma entrada indexada, no mesmo buffer.
   `Indexar` passou a ser atômico.
4. **Posse e rastro:** `Soltar` e `Acoplar` marcam o quadro como em trânsito;
   escrita em quadro solto recusa com `ErrSolto`. O rastro guarda os últimos
   8 saltos (`Marcas`), sem alocar.
5. **`pipeline.Juncao`:** tabela fixa de entidades pendentes; dispara a
   receita uma vez, quando as entradas fecham; cheia, recusa com `ErrCheio`
   (contrapressão); tag repetida é recusada sem perder o que já chegou.

## Decisões de desenho (ainda não implementadas)

1. **`Transmitir` é o único operador de passagem entre pipelines.** Sentido
   único, sem devolução. Desencaixa o espaço de memória da pipeline de origem;
   o Router o encaixa (`Acoplar`) na de destino. A pipeline de destino não
   tem `Recebi`. Recusa por falta de espaço (`ErrCheio`) não é devolução: o
   quadro nunca saiu e continua da origem. Depois de `Transmitir`, o `Seq`
   deve falhar com `ErrSolto` em qualquer passo seguinte.
2. **A transmissão é por campo (tipo), não por entidade.** O tipo que termina
   segue na hora para o tipo correspondente da struct da pipeline seguinte.
   O buffer é append-only e um campo escrito não muda, então leitores não
   precisam de lock; a transmissão só passa a posse, sem cópia. A única
   coordenação é a reciclagem: o buffer só volta ao pool quando a última
   região viva acabou (contador atômico).
3. **Receita é a função complexa redesenhada como `Seq` de átomos**, uma
   mini-thread interna. O átomo é a unidade de código; a receita é a unidade
   de execução. Uma pipeline é um **conjunto de receitas** sobre a mesma
   struct. Cada receita declara entradas e saídas e dispara sozinha quando as
   suas entradas estão presentes (AND de máscara).
4. **Divisão de responsabilidades:**
   - operadores e receitas: transformar o dado;
   - **adapter**: só o formato; garante que o dado chega ao destino no
     formato certo. É o único ponto aceitável de conversão (que copia), na
     borda; o zero-cópia vale do quadro para dentro;
   - **Router**: transporte e protocolos, `Acoplar`, tabela de entidades,
     medir e vigiar (fila, latência, rastro, processos críticos pendentes).

## Pendências (precisam de decisão)

1. **Quadro privado por execução de receita.** `Reg` mora no quadro, então
   duas receitas concorrentes sobre o mesmo quadro sobrescrevem os
   registradores uma da outra. Proposta: cada execução roda num quadro do
   pool, com registradores e saída próprios, ligado só às entradas que a
   receita declara. Sem aval ainda. Medir o custo com `-benchmem`.
2. **Corrida no `Ligar` atual.** `Campo` consulta o índice da origem
   (`faixas`, `presentes`) na hora da leitura. Isso só é seguro enquanto a
   origem não é mais tocada (entrega do quadro inteiro). No fluxo por campo,
   a origem continua indexando, e é corrida de dados. Correção: capturar a
   janela do campo no momento da transmissão. Escrever o teste de corrida
   (`-race`) antes de corrigir.
3. **`Juncao.Entregar` ainda executa a receita na goroutine de quem
   entrega.** No desenho novo ela só devolve a entidade completa, e o Router
   a entrega ao worker de destino. A estrutura deve ficar na biblioteca
   (pura); o uso e a vigilância são do Router.
4. **`MaxOrigens = 4`** não cobre 10 tipos por struct. Resolver junto com o
   item 1.
5. **Onde mora o adapter** e como se relaciona com `firewall/entrada` e
   `firewall/sanitizar`. Inclinação: `firewall/adapter`, separado.
6. **Entidade incompleta** não tem prazo: a tabela fixa só evita crescimento.
7. **Processos críticos** (os que dependem de variável externa): começar só
   com o mapa, sem prazo nem reenvio.
8. **Reenvio de tag para a mesma entidade:** hoje recusa (`ErrRepetida`). Se
   o reenvio legítimo precisar substituir, é uma mudança pequena.

## Consequências

- Primeiro teste a escrever, para a pendência 1: duas receitas concorrentes
  sobre os mesmos dados, cada uma com seus registradores, sob `-race`.
- O `TestCaminhoNaoAloca` continua valendo para o caminho de um quadro. O
  fluxo por campo precisa de um equivalente.
