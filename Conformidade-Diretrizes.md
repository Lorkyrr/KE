# Conformidade com as Diretrizes de Código

Como aplicar as [Diretrizes de Código](Diretrizes-de-Codigo.md) **2.2.1** a qualquer repositório. O checklist não muda com a linguagem. Mudam o comando que prova cada item e o que conta como erro descartado.

O `estrutura.sh` lê o bloco `diretrizes` do `README.md` (protocolo na seção *Protocolo do README* das diretrizes). Para cada papel sem comando no bloco, usa a tabela desta página. Grava `architecture-docs/Conformidade-Diretrizes.md` na raiz do projeto e monta o mesmo texto na seção *Conformidade aplicada* das diretrizes, entre os marcadores.

Se esta página e o script divergirem, vale o que o script executa. Os dois se atualizam no mesmo commit.

**Legenda do relatório gerado:** ✅ cumprido · 🟡 parcial · ❌ não cumprido · ⬜ revisão humana.

## O que o script confere sozinho

| Nível | Roda | Reprova a execução |
| --- | --- | --- |
| 1 | formatação, lint, varredura de segredo (`.env`, `.pem`, `.key`, `.p12`, `.pfx`, chave SSH; `.env.example` não conta) | comando que falha, ou segredo encontrado |
| 2 | mais testes, auditoria de dependência e dependência sem uso | o mesmo, mais teste ou auditoria que falham |
| 3 | mais cobertura e a seção `## Dados pessoais` | o mesmo, mais a seção ausente |
| 4 | os mesmos comandos | o mesmo, e também formatação, lint ou teste sem ferramenta |

Cobertura, dependência sem uso e auditoria manual (C e C++, sem ecossistema de pacotes) entram no relatório e não reprovam. Um comando declarado no README substitui o padrão daquele papel por inteiro; para manter dois, una-os com `&&`. `-` desliga o papel. A ferramenta precisa estar no `PATH`, com o ambiente do projeto ativo (virtualenv, `./gradlew`).

`gofmt -l` é o único comando cujo sucesso é saída vazia: se listar arquivo, reprova.

## Índice

| Id | Formatar | Lint | Testes | Auditoria |
| --- | --- | --- | --- | --- |
| `python` | ruff format | ruff check | pytest | pip-audit |
| `c` | clang-format | cppcheck (C) | ctest, se houver CTest | manual |
| `cpp` | clang-format | cppcheck (C++) | ctest, se houver CTest | manual |
| `java` | google-java-format | Checkstyle, se houver build | Maven ou Gradle | OWASP dependency-check, se houver build |
| `kotlin` | ktlint | detekt | Gradle | OWASP, se houver Gradle |
| `go` | gofmt | go vet e staticcheck | go test (com `-race` no nível 3+) | govulncheck |
| `rust` | cargo fmt | cargo clippy | cargo test | cargo audit |
| `typescript` | prettier | tsc | npm test | npm audit |
| `javascript` | prettier | eslint | npm test | npm audit |
| `bash` | shfmt | shellcheck | bats, se houver `.bats` | não se aplica |

## Python (`python`, `py`)

| Papel | Comando |
| --- | --- |
| Formatar | `ruff format --check .` |
| Lint | `ruff check .` |
| Testes | `pytest` |
| Auditoria | `pip-audit` |
| Dependência sem uso | `deptry .` |
| Cobertura | `pytest --cov` (plugin pytest-cov) |
| Ecossistema | `pip` |

Erro descartado: um `except` vazio, ou um `except Exception` que não trata, não registra e não relança.

Memória: coletor de lixo. Não é desvio.

## C (`c`)

| Papel | Comando |
| --- | --- |
| Formatar | `find` nos `*.c` e `*.h`, fora de `.git`, `node_modules`, `build`, `out` e `target`, e `xargs -r clang-format --dry-run --Werror` |
| Lint | `cppcheck --language=c --enable=warning,style,performance,portability --error-exitcode=1 --suppress=missingIncludeSystem .` |
| Testes | `ctest --test-dir build --output-on-failure` se existir `build/CTestTestfile.cmake`; `ctest --output-on-failure` se o arquivo estiver na raiz; senão o papel fica manual até o README declarar `testar:` |
| Auditoria | Manual. C não tem ecossistema de pacotes neste protocolo. Com lockfile, declare `auditar:` (por exemplo `osv-scanner --recursive .`) |
| Dependência sem uso | Manual. Não há gerenciador único |
| Cobertura | `gcovr -r .` |
| Ecossistema | `nenhum` |

Erro descartado: ignorar o retorno de uma função que sinaliza falha, ou seguir sem conferir `errno`.

Memória: gerência manual. As diretrizes só aceitam com ADR.

## C++ (`cpp`, `c++`)

| Papel | Comando |
| --- | --- |
| Formatar | O mesmo `clang-format` do C, também em `*.cc`, `*.cpp`, `*.cxx`, `*.hh`, `*.hpp` e `*.hxx` |
| Lint | `cppcheck --language=c++ --std=c++17 --enable=warning,style,performance,portability --error-exitcode=1 --suppress=missingIncludeSystem .` |
| Testes | A mesma regra do CTest |
| Auditoria | Manual, como em C |
| Dependência sem uso | Manual, como em C |
| Cobertura | `gcovr -r .` |
| Ecossistema | `nenhum` |

Erro descartado: `catch (...)` vazio, `catch` que descarta exceção, ou `std::error_code` / `std::expected` ignorado. Falha esperada não chama `std::abort`.

Memória: RAII e ponteiros inteligentes. `new`/`delete` solto exige ADR.

## Java (`java`)

Formatar, em qualquer layout: `find` nos `*.java` (fora de `.git`, `node_modules`, `build`, `out`, `target` e `.gradle`) e `xargs -r google-java-format --dry-run --set-exit-if-changed`. Para Spotless, declare `formatar: mvn -q spotless:check` ou o equivalente Gradle.

| Papel | Com `pom.xml` | Com `build.gradle` ou `build.gradle.kts` | Sem build |
| --- | --- | --- | --- |
| Lint | `mvn -q -DskipTests checkstyle:check` | `./gradlew checkstyleMain` (ou `gradle`, se não houver wrapper) | Manual. Declare `lint:` |
| Testes | `mvn -q test` | `./gradlew test` | Manual. Declare `testar:` |
| Auditoria | `mvn -q org.owasp:dependency-check-maven:check -DfailBuildOnCVSS=7` | `./gradlew dependencyCheckAnalyze` | Manual. Declare `auditar:` |
| Cobertura | `mvn -q test jacoco:report` | `./gradlew jacocoTestReport` | Manual. Declare `cobertura:` |
| Dependência sem uso | `mvn -q -DskipTests dependency:analyze` | Manual. Declare `dependencias-sem-uso:` | Manual |

O wrapper `./gradlew` ganha do `gradle` do sistema quando o arquivo existe. Checkstyle, JaCoCo e dependency-check precisam estar no build; se o objetivo não existir, declare o comando no README. `dependency:analyze` pode sair com status 0 mesmo listando dependência sem uso: leia a saída.

Erro descartado: `catch` vazio ou `catch (Exception)` que não trata, não registra e não relança.

Memória: coletor de lixo. Não é desvio.

Ecossistema: `maven` com `pom.xml`, `gradle` com Gradle, `nenhum` sem build.

## Kotlin (`kotlin`, `kt`)

| Papel | Com Gradle | Sem Gradle |
| --- | --- | --- |
| Formatar | `./gradlew ktlintCheck` | `ktlint` |
| Lint | `./gradlew detekt` | `detekt` |
| Testes | `./gradlew test` | Manual. Declare `testar:` |
| Auditoria | `./gradlew dependencyCheckAnalyze` | Manual. Declare `auditar:` |
| Cobertura | Manual. Declare `cobertura:` (Kover ou JaCoCo; o nome da task muda) | Manual |
| Dependência sem uso | Manual. Declare `dependencias-sem-uso:` | Manual |

Erro descartado: `runCatching` seguido de `getOrNull` ou `getOrDefault` que descarta a falha. Vale a regra do Java para `catch` vazio.

Memória: coletor de lixo. Não é desvio.

Ecossistema: `gradle`.

## Go (`go`, `golang`)

| Papel | Comando |
| --- | --- |
| Formatar | `gofmt -l .` — sucesso só com saída vazia |
| Lint | `go vet ./...` e `staticcheck ./...` |
| Testes | `go test ./...` no nível 2; `go test -race ./...` no nível 3 e no 4 |
| Auditoria | `govulncheck ./...` |
| Dependência sem uso | `go mod tidy -diff` |
| Cobertura | `go test -cover ./...` |
| Ecossistema | `gomod` |

Erro descartado: atribuir o erro a `_`, ou `panic` diante de arquivo, rede ou entrada do usuário.

Memória: coletor de lixo. Não é desvio.

## Rust (`rust`, `rs`)

| Papel | Comando |
| --- | --- |
| Formatar | `cargo fmt --all -- --check` |
| Lint | `cargo clippy --all-targets --all-features -- -D warnings` |
| Testes | `cargo test --all-features` |
| Auditoria | `cargo audit` |
| Dependência sem uso | `cargo machete` |
| Cobertura | `cargo llvm-cov --summary-only` |
| Ecossistema | `cargo` |

Erro descartado: `unwrap` ou `expect` sobre arquivo, entrada do usuário ou saída de outro programa.

Memória: ownership e borrow checker. Não é desvio.

Os anexos E a H das diretrizes são este conjunto, escrito como hook e CI.

## TypeScript (`typescript`, `ts`) e JavaScript (`javascript`, `js`)

| Papel | TypeScript | JavaScript |
| --- | --- | --- |
| Formatar | `npx --no-install prettier --check .` | o mesmo |
| Lint | `npx --no-install tsc --noEmit` | `npx --no-install eslint .` |
| Testes | `npm test` | `npm test` |
| Auditoria | `npm audit --audit-level=high` | o mesmo |
| Dependência sem uso | `npx --no-install knip` | o mesmo |
| Cobertura | Manual. Declare `cobertura:` | Manual |
| Ecossistema | `npm` | `npm` |

`--no-install` não baixa ferramenta que o projeto não fixou. Com as duas linguagens no mesmo README, o script não repete o comando idêntico.

Erro descartado em TypeScript: `catch` vazio. Na borda do sistema, `any` não substitui validação.

Erro descartado em JavaScript: `catch` vazio. Dado externo se valida antes de seguir.

Memória: coletor de lixo. Não é desvio.

## Bash (`bash`, `sh`)

| Papel | Comando |
| --- | --- |
| Formatar | `shfmt -d .` |
| Lint | `find` nos `*.sh` e `xargs -r shellcheck` |
| Testes | `bats .` se existir arquivo `.bats`; senão manual |
| Auditoria | Não se aplica. Bash não declara dependência de pacote neste protocolo |
| Dependência sem uso | Não se aplica |
| Cobertura | Não se aplica |
| Ecossistema | `nenhum` |

Erro descartado: comando cujo status é ignorado, script sem `set -euo pipefail`, ou erro mandado para `/dev/null` sem tratamento.

Memória: não se aplica.

## Itens que continuam humanos

O script não marca como cumprido o que exige leitura de código: classificação do erro, validação de entrada, confirmação de ação destrutiva, estados da interface, TLS, autenticação, retry, idempotência, fila, timeout, log, classificação offline, SLO, DLQ, auditoria de acesso e os três cenários de falha. Esses itens saem ⬜, com o critério da linguagem na coluna de evidência quando ele depende dela (erro descartado e memória).

No nível 3 e no 4, a seção `## Dados pessoais` é obrigatória mesmo quando a única linha é "nenhum". A chave `dados-pessoais` (`sim` ou `nao`) não substitui a seção.

## Exemplo de documentação e de estrutura

O KEngine entra só como forma. Especificação de produto não fica neste arquivo.

Documentação, em qualquer projeto:

| Peça | Caminho |
| --- | --- |
| README no protocolo | `README.md` |
| Changelog | `CHANGELOG.md` |
| Decisão e desvio | `docs/adr/0000-modelo.md` e os números seguintes |
| Pedido de mudança | `.github/pull_request_template.md` |
| Conformidade gerada | `architecture-docs/Conformidade-Diretrizes.md` |
| Specs e épicos | `.planning/specs/`, `.planning/epics/` |

Estrutura que o `estrutura.sh` garante, com o mesmo nome em todo repositório:

```text
.github/workflows/
.githooks/
docs/adr/
architecture-docs/
.planning/specs/
.planning/epics/
```

Código, módulo e build continuam onde a linguagem já os coloca. O script não cria essa parte e não substitui arquivo que já existe.
