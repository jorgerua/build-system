# Documento de Requisitos: Melhoria de Testes Unitários e Integrados

## Introdução

Este documento define os requisitos para melhorar a cobertura de testes unitários (meta de ~80%) e corrigir os testes integrados do OCI Build System. Os testes integrados devem ser executados via Makefile, com o ambiente Docker Compose funcionando corretamente, incluindo a correção de bugs de configuração como o `GITHUB_WEBHOOK_SECRET`.

## Glossário

- **Unit_Test**: Teste que verifica o comportamento de uma unidade isolada de código (função, método, struct)
- **Integration_Test**: Teste que verifica a interação entre múltiplos componentes do sistema
- **Code_Coverage**: Percentual de linhas de código executadas durante os testes
- **Test_Suite**: Conjunto de testes relacionados a um componente ou funcionalidade
- **Docker_Compose**: Ferramenta para definir e executar aplicações multi-container
- **Makefile**: Arquivo de automação de tarefas de build e teste
- **Property_Based_Test**: Teste que verifica propriedades universais através de múltiplas entradas geradas

## Requisitos

### Requisito 1: Cobertura de Testes Unitários

**User Story:** Como desenvolvedor, eu quero que o código tenha cobertura de testes próxima a 80%, para que eu tenha confiança na qualidade e corretude do sistema.

#### Critérios de Aceitação

1. WHEN testes unitários são executados com flag de cobertura, THE System SHALL reportar cobertura de código de pelo menos 75%
2. THE System SHALL ter testes unitários para todos os handlers HTTP (webhook, status, health)
3. THE System SHALL ter testes unitários para todos os middlewares (auth, logging)
4. THE System SHALL ter testes unitários para todas as operações de Git Service (clone, pull, sync)
5. THE System SHALL ter testes unitários para todas as operações de Cache Service (init, clean, get size)
6. THE System SHALL ter testes unitários para todas as operações de Image Service (build, tag)
7. THE System SHALL ter testes unitários para o Worker Service orchestrator
8. THE System SHALL ter testes unitários para o NATS Client (connect, publish, subscribe)
9. WHEN um teste unitário falha, THE System SHALL fornecer mensagem de erro clara indicando o que falhou e por quê
10. THE System SHALL ter testes para casos de erro e edge cases, não apenas happy path

### Requisito 2: Correção de Bugs de Configuração

**User Story:** Como desenvolvedor, eu quero que o ambiente Docker Compose suba corretamente com todas as configurações necessárias, para que eu possa executar testes integrados sem problemas.

#### Critérios de Aceitação

1. WHEN docker-compose up é executado, THE System SHALL carregar corretamente a variável GITHUB_WEBHOOK_SECRET do ambiente ou usar valor padrão
2. THE System SHALL validar que todas as variáveis de ambiente obrigatórias estão definidas antes de iniciar os serviços
3. WHEN uma variável de ambiente obrigatória está ausente, THE System SHALL falhar com mensagem de erro clara indicando qual variável está faltando
4. THE System SHALL carregar configurações de arquivos config.yaml com suporte a substituição de variáveis de ambiente usando sintaxe ${VAR_NAME}
5. WHEN um serviço é iniciado, THE System SHALL registrar no log as configurações carregadas (sem expor secrets)
6. THE System SHALL ter um arquivo .env.example documentando todas as variáveis de ambiente necessárias
7. THE System SHALL validar que os caminhos de cache configurados existem e são graváveis na inicialização

### Requisito 3: Execução de Testes Integrados via Makefile

**User Story:** Como desenvolvedor, eu quero executar testes integrados através de comandos make simples, para que o processo de teste seja padronizado e fácil de usar.

#### Critérios de Aceitação

1. THE System SHALL ter um comando `make test-integration` que executa todos os testes integrados
2. WHEN `make test-integration` é executado, THE System SHALL primeiro subir o ambiente Docker Compose
3. WHEN o ambiente Docker Compose está pronto, THE System SHALL executar os testes integrados
4. WHEN os testes integrados são concluídos, THE System SHALL fazer cleanup do ambiente Docker Compose
5. THE System SHALL ter um comando `make test-integration-setup` que apenas sobe o ambiente sem executar testes
6. THE System SHALL ter um comando `make test-integration-teardown` que apenas derruba o ambiente
7. WHEN testes integrados falham, THE System SHALL preservar logs dos containers para análise
8. THE System SHALL ter um timeout configurável para aguardar que os serviços estejam prontos antes de executar testes
9. WHEN um serviço não fica pronto dentro do timeout, THE System SHALL falhar com mensagem clara indicando qual serviço não respondeu

### Requisito 4: Health Checks e Readiness

**User Story:** Como desenvolvedor, eu quero que os serviços tenham health checks adequados, para que os testes integrados só sejam executados quando o ambiente estiver realmente pronto.

#### Critérios de Aceitação

1. THE API Service SHALL expor endpoint `/health` que retorna 200 quando o serviço está pronto
2. THE API Service health check SHALL verificar conectividade com NATS antes de retornar sucesso
3. THE Worker Service SHALL ter health check que verifica conectividade com NATS
4. WHEN um serviço não consegue conectar ao NATS, THE health check SHALL retornar status não-pronto
5. THE Docker Compose configuration SHALL ter health checks configurados para todos os serviços
6. THE Makefile test-integration command SHALL aguardar que todos os health checks estejam passando antes de executar testes
7. THE System SHALL ter um script de wait-for-services que verifica health checks com retry e timeout

### Requisito 5: Validação de Testes Antes de Correção de Código

**User Story:** Como desenvolvedor, eu quero que o sistema valide se um teste está correto antes de assumir que o código tem um bug, para evitar correções desnecessárias.

#### Critérios de Aceitação

1. WHEN um teste falha, THE Developer SHALL primeiro revisar o teste para verificar se ele está testando o comportamento correto
2. WHEN um teste falha, THE Developer SHALL verificar se o teste tem mocks ou stubs configurados corretamente
3. WHEN um teste falha, THE Developer SHALL verificar se o teste tem assertions corretas e mensagens de erro claras
4. WHEN um teste integrado falha, THE Developer SHALL verificar se o ambiente está configurado corretamente (variáveis de ambiente, volumes, network)
5. WHEN múltiplos testes falham no mesmo componente, THE Developer SHALL considerar que pode ser um problema no código, não nos testes
6. THE System SHALL ter documentação clara sobre como debugar testes que falham
7. THE System SHALL ter exemplos de testes bem escritos para cada tipo de componente

### Requisito 6: Testes de Propriedades (Property-Based Tests)

**User Story:** Como desenvolvedor, eu quero que propriedades críticas do sistema sejam validadas através de property-based testing, para garantir corretude em múltiplos cenários.

#### Critérios de Aceitação

1. THE System SHALL ter property-based tests para validação de assinatura de webhook (Propriedade 1)
2. THE System SHALL ter property-based tests para extração de informações de webhook (Propriedade 2)
3. THE System SHALL ter property-based tests para detecção automática de linguagem (Propriedade 11)
4. THE System SHALL ter property-based tests para aplicação de tags de imagem (Propriedade 15)
5. WHEN um property-based test falha, THE System SHALL reportar o caso de teste que causou a falha (shrinking)
6. THE System SHALL executar pelo menos 100 iterações para cada property-based test
7. THE System SHALL ter property-based tests anotados com comentário indicando qual propriedade do design está sendo validada

### Requisito 7: Organização e Estrutura de Testes

**User Story:** Como desenvolvedor, eu quero que os testes sejam bem organizados e fáceis de encontrar, para facilitar manutenção e adição de novos testes.

#### Critérios de Aceitação

1. THE System SHALL ter testes unitários co-localizados com o código fonte usando sufixo `_test.go`
2. THE System SHALL ter testes de propriedade co-localizados com o código fonte usando sufixo `_property_test.go`
3. THE System SHALL ter testes integrados em diretório separado `tests/integration/`
4. THE System SHALL ter fixtures de teste (repositórios de exemplo) em `tests/integration/fixtures/`
5. THE System SHALL ter helpers e utilitários de teste em package `testutil`
6. WHEN um novo componente é adicionado, THE System SHALL ter testes criados junto com o componente
7. THE System SHALL ter convenção de nomenclatura clara para testes: `Test<ComponentName>_<Behavior>`

### Requisito 8: Relatórios de Teste

**User Story:** Como desenvolvedor, eu quero relatórios claros dos resultados de testes, para identificar rapidamente o que falhou e por quê.

#### Critérios de Aceitação

1. WHEN testes unitários são executados, THE System SHALL gerar relatório de cobertura em formato HTML
2. WHEN testes integrados são executados, THE System SHALL gerar relatório com logs de cada teste
3. THE System SHALL ter comando `make test-report` que abre o relatório de cobertura no browser
4. WHEN um teste falha, THE System SHALL incluir no relatório: nome do teste, mensagem de erro, stack trace, e tempo de execução
5. THE System SHALL ter relatório consolidado mostrando: total de testes, passados, falhos, skipped, e cobertura
6. THE System SHALL preservar relatórios de testes anteriores em diretório `test-reports/` com timestamp
7. WHEN testes são executados em CI, THE System SHALL gerar relatórios em formato compatível com ferramentas de CI (JUnit XML)

### Requisito 9: Performance de Testes

**User Story:** Como desenvolvedor, eu quero que os testes sejam executados rapidamente, para ter feedback rápido durante o desenvolvimento.

#### Critérios de Aceitação

1. WHEN testes unitários são executados, THE System SHALL completar em menos de 30 segundos
2. WHEN testes integrados são executados, THE System SHALL completar em menos de 5 minutos
3. THE System SHALL suportar execução paralela de testes unitários usando flag `-parallel`
4. THE System SHALL ter testes lentos marcados com build tag `//go:build slow` para execução opcional
5. THE System SHALL usar mocks para dependências externas em testes unitários para melhorar velocidade
6. WHEN testes integrados são executados, THE System SHALL reutilizar containers Docker quando possível
7. THE System SHALL ter comando `make test-quick` que executa apenas testes rápidos

### Requisito 10: Documentação de Testes

**User Story:** Como desenvolvedor, eu quero documentação clara sobre como escrever e executar testes, para facilitar contribuições e manutenção.

#### Critérios de Aceitação

1. THE System SHALL ter documento TESTING.md explicando estratégia de testes
2. THE TESTING.md SHALL documentar como executar testes unitários, integrados, e de propriedade
3. THE TESTING.md SHALL documentar como gerar relatórios de cobertura
4. THE TESTING.md SHALL ter exemplos de como escrever cada tipo de teste
5. THE TESTING.md SHALL documentar como debugar testes que falham
6. THE TESTING.md SHALL documentar requisitos de ambiente para testes integrados
7. THE System SHALL ter comentários em testes complexos explicando o que está sendo testado e por quê
