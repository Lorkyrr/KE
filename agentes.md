# Agentes

Quem constrói o quê, com qual modelo e com qual esforço.
Lido em 07/10/2026 a partir da documentação do Grok
(`https://docs.x.ai/developers/model-capabilities/text/reasoning`,
`https://docs.x.ai/build/settings`), do Grok Bot
(`https://docs.x.ai/grok-bot/overview`) e do Claude Code
(`https://code.claude.com/docs/en/model-config`,
`https://code.claude.com/docs/en/cross-session-messaging`).

Este arquivo não redecide produto. Spec, ADR e regra de negócio ficam no
repositório do projeto. Pergunta sem resposta escrita continua sem
resposta. A árvore de documentação que todo projeto compartilha está em
[Conformidade-Diretrizes.md](Conformidade-Diretrizes.md).

## 1. Três agentes, três lugares

| Agente | Onde roda | O que faz |
| --- | --- | --- |
| **Grok Build** (Grok 4.7, esta família de sessão) | Terminal, no worktree | Lê o que já está no repositório, fecha a seção do spec que diz onde a mudança entra, revisa o diff |
| **Claude Code** (Haiku 5.5, Sonnet 5.5 ou Opus 5.5) | Terminal, no worktree | Implementa. Haiku no mecânico. Sonnet no escopo fechado. Opus na invariante |
| **Grok Bot** | Uma VM na nuvem da Cursor, compartilhada por todos os Bots da conta | Olha site logado e rotina no horário. Não implementa e não guarda segredo |

O Grok Bot não escolhe modelo nem esforço. O computador é um por conta:
login, cookie e arquivo de um Bot ficam visíveis para os outros. Isolar
pelo nome do Bot não isola credencial. Senha, `.env`, chave privada,
certificado e token do GitHub com escrita não entram nessa VM.

Cursor Cloud Agent, quando o Bot delega código, é outro computador, sob o
controle de Cloud Agents do time. Não é este worktree e não é o Claude
Code. O admin pode desligar essa delegação.

## 2. Os cinco esforços

O mesmo nome não custa a mesma coisa nos dois produtos. A escala de cada
um vale só dentro dele.

Claude Code, em
`https://code.claude.com/docs/en/model-config`. O seletor existe no
Haiku 5.5, no Sonnet 5.5, no Opus 5.5 e nos modelos a partir de
Sonnet 4.6 e Opus 4.6. Haiku anterior ao 5.5 não entra nesta tabela: sem
seletor, o mecânico vai para o Sonnet 5.5 em `low`.

| Nível | Quando a doc manda usar |
| --- | --- |
| `low` | Troca curta, com alguém revendo cada resultado: rascunho, renome, ajuste de uma linha |
| `medium` | Padrão do Haiku 5.5, do Sonnet 5.5 e do Opus 5.5. Engenharia do dia com escopo fechado |
| `high` | Verificação e caso de borda, como bug num código que já existe. Padrão dos outros modelos com seletor, fora o Opus 4.7 |
| `xhigh` | Raciocínio mais fundo, mais token. Padrão do Opus 4.7 |
| `max` | Problema duro, sessão sem a pessoa no laço. Vale só nesta sessão, salvo `CLAUDE_CODE_EFFORT_LEVEL`. Retorno diminui e o modelo enrola |

Grok 4.7, em
`https://docs.x.ai/developers/model-capabilities/text/reasoning`. O
raciocínio não desliga. `xhigh` existe no Grok 4.6 e nos seguintes.

| Nível | Quando a doc manda usar |
| --- | --- |
| `low` | Chamada de ferramenta simples, resposta que precisa ser rápida |
| `medium` | Análise e contexto longo |
| `high` | Problema duro, vários passos, lógica sem atalho. **É o valor da API quando o campo vem vazio** |
| `xhigh` | Teto de profundidade. Mais latência e mais custo |

Medido no Artificial Analysis em outubro de 2026: Opus 5.5 em `medium`
marca 51 e custa menos por tarefa que o Grok 4.7 em `high` (46).
Terminal-Bench 4.0: Opus cerca de 60%, Sonnet 5.5 cerca de 64%, Grok 4.7
cerca de 25%. Subir o Grok de `high` para `xhigh` deixa o índice em 46 e
encarece. Fluxo longo de terminal (teste, comando, conserto) fica no
Claude. Leitura de spec e revisão de diff ficam no Grok, no nível da §3,
que não é `high`.

## 3. Pedido, modelo, esforço

Uma linha por pedido. O esforço vai no parâmetro da §4. Sessão que não
nomeia o nível herda o padrão do produto: no Grok 4.7 isso é `high`, e é
o desperdício que esta tabela corta.

| Pedido | Quem | Modelo | Esforço |
| --- | --- | --- | --- |
| Achar arquivo, citar um trecho, responder onde algo está, escrever a linha da roadmap | Grok Build | Grok 4.7 | `low` |
| Ler README, spec ou ADR; escrever a seção de entrada do spec; revisar diff | Grok Build | Grok 4.7 | `medium` |
| Lógica em vários passos sem resposta escrita no repositório, ou o `medium` falhou na mesma pergunta | Grok Build | Grok 4.7 | `high` |
| O `high` falhou e a pessoa pediu o teto | Grok Build | Grok 4.7 | `xhigh` |
| Renomear, formatar, uma linha, changelog, fixture com o contrato já escrito, linha da roadmap | Claude Code | Haiku 5.5 | `low` |
| Implementar escopo fechado no spec, teste que segue contrato já escrito, tela sem regra nova | Claude Code | Sonnet 5.5 | `medium` |
| Invariante, estado, porta, dinheiro, migração que atravessa mais de um módulo | Claude Code | Opus 5.5 | `medium` |
| A mesma invariante falhou duas vezes | Claude Code | Opus 5.5 | `high` |
| Sessão longa sem a pessoa no laço, ou auditoria de segurança pedida | Claude Code | Opus 5.5 | `xhigh` |
| A pessoa pediu o teto do Claude nesta sessão | Claude Code | Opus 5.5 ou Sonnet 5.5 | `max` |
| Site logado, rotina, laptop fechado | Grok Bot | sem escolha | sem parâmetro |

`medium` é o padrão do Grok Build nesta casa. `high` é exceção com motivo
na tabela. `xhigh` e `max` não são o dia a dia.

Haiku não decide regra. Opus não faz o mecânico: desenha a porta, a
regra, o estado, a invariante e a classificação de erro, revisa, e passa
o resto a um subagente.

O subagente sai pela ferramenta `Agent` com `model: "haiku"`. O esforço
dele é `low`. Se a ferramenta tiver campo de esforço, o valor é `low`.
Um subagente por vez no mesmo worktree. O prompt é fechado: arquivos que
ele pode mexer, o contrato já pronto (tipos e assinaturas), o comando de
teste que tem de passar, o que não tocar. A sessão que delegou lê o diff
e roda os testes. O subagente não decide regra.

Quando o julgamento ainda é necessário no miúdo (vários arquivos, com
escolha), o subagente é Sonnet 5.5 em `medium`, um por vez, com o mesmo
prompt fechado. Se o Kepler recusar `modelId` `haiku`, a chamada usa
`sonnet` com esforço `low` e a resposta registra a recusa. Não sobe para
Sonnet em `high` no lugar.

Cabe no Haiku em `low`: teste e fixture de contrato já escrito, tela sem
regra nova, binding e rota que só repassam, adapter falso cujo contrato
já está escrito, `CHANGELOG.md`, épico, spec.

## 4. Onde o esforço é gravado

Cada chamada preenche um destes campos. Os nomes não se misturam: esforço
do Claude não vale na sessão do Grok, e o contrário também.

Claude Code, sessão Kepler (`create_session`):

```text
agentId: claude-code
modelId: haiku | sonnet | opus
configOverrides.effort: low | medium | high | xhigh | max
modeId: plan | acceptEdits
```

Fora do Kepler: `/effort <nível>`, `claude --effort <nível>`, ou a
variável `CLAUDE_CODE_EFFORT_LEVEL`. Para durar entre sessões,
`effortLevel` ou `modelSettings.<modelo>.effortLevel` nas settings
(Claude Code v2.1.251 ou mais novo). `max` não entra em settings; só na
sessão, ou pela variável de ambiente. Um subagente de arquivo pode trazer
`effort` no frontmatter.

Grok Build. O campo vazio na API é `high`. Por isso a sessão não fica
nesse nível entre um pedido e outro.

- Pedido atual: `/effort low`, `/effort medium`, `/effort high` ou
  `/effort xhigh`. No fim do pedido que não é `medium`, a sessão volta
  para `/effort medium`.
- Processo: `grok --effort medium` (o mesmo que `--reasoning-effort`).
- Sessão nova, para o esquecimento não cair no padrão da API, em
  `~/.grok/config.toml`:

```toml
[models]
default_reasoning_effort = "medium"
```

- Subagente herda o esforço da sessão pai. Persona com `reasoning_effort`
  ganha da sessão. A ferramenta de spawn desta CLI não recebe esforço no
  argumento: ou a sessão já está no nível da tabela, ou a persona traz o
  campo.
- Workflow: o lançamento manda `effort`. A opção `effort` do script ganha
  do lançamento. Um workflow sem os dois acompanha a sessão, então a
  sessão precisa já estar no nível certo.
- API direta: `reasoning_effort` (também `reasoning.effort`) com `low`,
  `medium`, `high` ou `xhigh`. Sem o campo, a API usa `high`.
- ACP: `configId` `reasoning_effort`.

Grok Bot: não há campo. Quem escreve o prompt da rotina descreve o
entregável em texto. Não pede modelo.

## 5. Protocolos que existem

Não há um protocolo único que os três falem. Cada produto aciona os seus.

### 5.1 Kepler — Grok Build e Claude Code nesta tarefa

O servidor MCP `kepler-workspace` é o acionamento que esta tarefa já tem.
Vale para a sessão que o Kepler sobe. Um `claude` aberto no terminal,
fora do Kepler, não recebe esse MCP só por estar no mesmo diretório.

Ferramentas, e o que cada uma faz:

- `create_session` abre outra sessão na mesma tarefa. Para o Claude, os
  campos são os da §4, com `modelId` e `configOverrides.effort` copiados
  da linha da §3. `modeId` fica em `plan` enquanto a seção de entrada do
  spec não existe, e em `acceptEdits` depois. Nomear `agentId`, `modelId`
  ou `modeId` troca o harness inteiro para o padrão desse agente.
- `session_send` manda um follow-up. Na mesma tarefa a entrega é
  imediata. Para outra tarefa o Kepler devolve antes quem seria
  contactado: a pessoa confirma, e só então `confirm: true`.
- `session_read` lê a resposta. O `create_session` devolve um handle
  `kepler-session:…`. A mensagem volta quando a outra sessão responde,
  não quando o envio é aceito.
- Não envie para a própria sessão.

Os dois não editam o mesmo worktree ao mesmo tempo. Sessão sequencial:
um para, o outro escreve, no mesmo worktree. Sessão paralela: antes,
`create_worktree` numa branch nova, e o `create_session` recebe esse
`worktreeId`. A junção é o PR.

O Grok Build não aparece na lista de peers do Claude Code. O caminho
Grok → Claude nesta tarefa é o MCP acima, não o `SendMessage`.

### 5.2 Claude Code — uma sessão Claude para outra

A partir do Claude Code v2.1.224 no Linux (v2.1.234 no Windows nativo),
uma sessão Claude lista as outras com `ListAgents` e fala com elas com
`SendMessage`. Na mesma máquina a mensagem vai por um socket Unix
(`CLAUDE_CODE_MESSAGING_SOCKET`), sem passar pelos servidores da
Anthropic. Documento:
`https://code.claude.com/docs/en/cross-session-messaging`.

Limites que importam aqui:

- O destino é outra sessão Claude. Não é o Grok Build e não é o Grok Bot.
- A mensagem é texto. Não leva o histórico nem autoriza nada. Não responde
  prompt de permissão, não muda `CLAUDE.md` nem configuração, e um comando
  escrito nela não executa.
- A sessão que recebe continua sujeita às permissões dela.
- Dois checkouts em worktrees diferentes podem se avisar do que já
  aterrisou. O aviso aponta o caminho do spec. Não pede para o outro
  reimplementar.

Use isto quando duas sessões Claude estiverem abertas no mesmo
repositório e uma descobrir um fato que a outra precisa no meio da tarefa.

### 5.3 Grok Bot — um Bot para outro, e daí para um Cloud Agent

Documentação: `https://docs.x.ai/grok-bot/overview`,
`https://docs.x.ai/grok-bot/chat-and-collaboration`,
`https://docs.x.ai/grok-bot/teams-and-enterprises`.

- Mensagem direta assíncrona: o Bot que recebe acorda, faz a parte dele
  e pode responder depois. A pessoa vê o handoff.
- Grupo de 2 a 6 Bots, com um resultado só e o dono do próximo passo
  nomeado. Handoff dentro do grupo é texto. Imagem vai em mensagem
  direta.
- Rotina: um Bot, um horário, uma skill já ensaiada uma vez na frente da
  pessoa. Até 50 rotinas por Bot.
- Delegação de código: o Bot pode pedir um Cursor Cloud Agent, em outro
  computador, se o time não tiver desligado "Cloud Agents". O resultado
  desse agente não cai neste worktree. Trazer para o repositório é um PR
  lido por uma sessão Kepler, não um commit feito da VM.

O Bot não tem `create_session` do Kepler e o Claude Code não lista Bots
em `ListAgents`. Não há ponte pronta entre os três. O arquivo no git é
a ponte.

### 5.4 O que não usar como ponte

- Apontar o Claude Code para a API do Grok. O loop do Claude fala o
  formato de mensagens da Anthropic. O Grok 4.7 na API é outro contrato.
- Postar no socket `CLAUDE_CODE_MESSAGING_SOCKET` a partir de um processo
  que não é filho da sessão Claude. O socket existe para peers Claude e
  para hook ou comando da própria sessão. Um Grok solto não é peer.
- Hook `Stop` que chama outro agente para continuar sozinho. A pessoa
  perde o ponto em que aprovaria o spec, e os dois passam a editar sem
  worktree combinado.
- Protocolo genérico de agente (A2A ou semelhante). As docs destes três
  produtos não o usam entre si.

## 6. Protocolo do repositório

O contrato é o spec. A mensagem só aponta o caminho. Quem recebe não
reexplora o repositório e não inventa regra que o spec deixou em aberto.

Caminhos, os mesmos em qualquer projeto:

| Peça | Caminho |
| --- | --- |
| README no protocolo | `README.md` |
| Changelog | `CHANGELOG.md` |
| Decisão | `docs/adr/` |
| Conformidade gerada | `architecture-docs/Conformidade-Diretrizes.md` |
| Specs e épicos | `.planning/specs/`, `.planning/epics/` |
| Roadmap (o que já foi feito) | `.planning/ROADMAP.md` |
| Diretrizes | `Diretrizes-de-Codigo.md` |

1. **Grok Build**, em `medium`, escreve ou completa a seção do spec que
   diz onde a mudança entra. Se o spec marca a mudança como fora de
   escopo, para e lista a pergunta. Não abre sessão nenhuma.
2. Com o spec aprovado pela pessoa e o pedido explícito de implementar,
   o Grok Build chama `create_session` na mesma tarefa. `modelId` e
   `configOverrides.effort` saem da §3. Worktree desta sessão se o Grok
   não for editar ao mesmo tempo. O prompt é o da §8.
3. **Claude Code** implementa o que essa seção cita. Teste antes do
   código quando as diretrizes do repositório pedem. Não abre caminho
   fora da lista. No fim, a resposta cabe em: arquivos mexidos, comando
   de teste e saída, o que ficou de fora.
4. **Grok Build**, em `medium`, lê essa sessão com `session_read` e o
   diff, contra o spec. Não reimplementa. Achou invariante de estado ou
   de valor: uma mensagem curta de volta (`session_send`) com o arquivo
   e o teste que falta. A pessoa vê as duas sessões. Essa revisão não
   sobe o esforço para `high`.
5. Duas sessões Claude no mesmo repositório se avisam por `SendMessage`
   com o caminho do arquivo que mudou. Não colam patch na mensagem.
6. **Grok Bot** entra para site logado, rotina e laptop fechado. O
   entregável volta como texto para a pessoa (ou um link). Esta sessão,
   em `low` ou `medium` conforme a §3, transforma isso em nota no spec.
   O Bot não faz merge e não comenta em PR.

Ninguém atualiza a tabela do épico nem numera ADR sem `git fetch` e a
conferência do repositório. Sessões paralelas colidem nesses números.
A exceção obrigatória é a linha da roadmap, na §7: entra no fechamento
da feature e não espera o processo de documentação.

## 7. Fechamento da feature

A feature não está pronta enquanto `.planning/ROADMAP.md` não aponta o
que ficou feito. Esse arquivo é o índice que a próxima sessão lê antes
de reexplorar o repositório. Não substitui spec, ADR, changelog nem
`architecture-docs/`. Reescrever essa documentação é outro processo, com
pedido próprio.

Quem declara a feature pronta escreve a linha. Se houve revisão, é a
sessão que leu o diff e aceitou. Se a sessão que implementou está sozinha,
é ela. As outras não reescrevem a mesma linha.

A linha entra no commit da feature, ou no commit seguinte da mesma
branch, antes do PR. Roadmap só no worktree não serve para as outras
sessões. `git fetch` antes de editar. Mexa na linha desta feature.

Se o arquivo não existe, crie com o cabeçalho abaixo e uma linha. Não
invente linha para trabalho antigo.

```markdown
# Roadmap

Índice do que já entrou. O spec continua em `.planning/specs/`. Esta
página não copia regra nem decisão.

| Estado | Feature | Spec | Onde ficou |
| --- | --- | --- | --- |
| feita | <nome> | `.planning/specs/<arquivo>.md` | <data AAAA-MM-DD>. <branch ou PR>. Teste: `<comando>` passou. Fora: <lista ou nenhuma> |
```

Estado é `feita`, `em curso` ou `não feita`. No fechamento, a linha
desta feature fica `feita`. Uma frase no "Onde ficou": data, branch ou
PR, comando de teste que passou, o que ficou de fora. Sem corpo de spec,
sem número de ADR novo, sem pacote.

Esforço deste passo: `low` (§3). No Grok, `/effort low` para a linha e,
ao terminar, `/effort medium`.

### Compactar a conversa e o cache

Depois que a linha está no commit, cada sessão que trabalhou a feature
compacta a si mesma, antes de pegar outra. A próxima sessão se acha pela
roadmap, não pelo histórico longo.

O foco do compacto é curto e igual nos dois:

```text
Feature <nome> feita. Roadmap: .planning/ROADMAP.md. Spec: <caminho>. Arquivos: <lista>. Teste: <comando> passou. Fora: <lista ou nenhuma>.
```

- **Grok Build:** `/flush` e, em seguida, `/compact` com esse foco.
  `/flush` grava na memória o que o compacto descartaria. `/compact`
  comprime o histórico da conversa. Não é `/compact-mode` (isso só
  aperta a tela) e não é `/clear` (isso abre sessão vazia e perde o
  ponteiro). Não apague `~/.grok` nem cache de build do projeto.
- **Claude Code:** `/compact` com o mesmo foco. O comando substitui o
  histórico por um resumo e invalida o cache da conversa; o turno
  seguinte reconstrói o cache só com esse resumo
  (`https://code.claude.com/docs/en/prompt-caching`). `/clear` fica para
  quando a próxima tarefa não tem relação com esta. Não compacte antes
  da linha da roadmap estar no commit: o resumo precisa citá-la.

A sessão que implementou, se ainda estiver aberta e não for quem escreve
a roadmap, espera o aviso com o caminho `.planning/ROADMAP.md` e aí
compacta. Não cola patch nesse aviso.

## 8. Mensagem que aciona o Claude

O `create_session` leva só isto, com os campos preenchidos. Modelo e
esforço repetem a §3, para a sessão não cair no padrão dela.

```text
Pedido: <nome>.
Spec: .planning/specs/<arquivo>.md
Contrato: <caminho da seção no spec ou no ADR>. Não abra arquivo fora da lista.
Modelo: <haiku|sonnet|opus>. Esforço: <low|medium|high|xhigh|max>.
Perguntas em aberto que você não responde: <lista ou "nenhuma">.
O mecânico vai a subagente Haiku em low (agentes.md §3). Sonnet em medium só se o Haiku não couber.
Não reescreva a documentação. Quem declara a feature pronta atualiza só .planning/ROADMAP.md (agentes.md §7) e, depois do commit dessa linha, compacta a conversa.
Devolva só: arquivos mexidos, comando de teste e a saída, o que ficou de fora.
```

Modo `plan` se a seção de entrada ainda não está no spec. `acceptEdits`
quando está.

## 9. O que não entra neste arquivo

Catálogo de feature, número de ADR, nome de pacote, código de cliente,
adquirente e ambiente de um produto. Isso vive no `.planning/` e no
`docs/adr/` daquele repositório. A roadmap também: cada projeto tem a
sua em `.planning/ROADMAP.md`. Este guia só aponta o caminho, escolhe
modelo e esforço, e manda fechar a feature por essa linha.
