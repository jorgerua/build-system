# Plano de Implementação: OCI Build System

## Visão Geral

Este plano implementa um sistema de build OCI distribuído usando Go, NATS, NX, e buildah. A implementação segue uma arquitetura de microserviços com comunicação via message broker, organizada em um monorepo gerenciado por NX.

## Tarefas

- [x] 1. Configurar estrutura do monorepo e dependências base
  - Criar estrutura de diretórios (apps/, libs/, tests/)
  - Inicializar NX workspace
  - Configurar nx.json com targets de build e test
  - Criar go.mod na raiz do projeto
  - Adicionar dependências: gin, fx, zap, viper, nats.go, go-git
  - Criar docker-compose.yml com NATS
  - _Requisitos: 10.1_

- [x] 2. Implementar biblioteca compartilhada (libs/shared)
  - [x] 2.1 Criar tipos de dados compartilhados
    - Definir structs: BuildJob, JobStatus, PhaseMetric, RepositoryInfo
    - Definir enums: JobStatus, BuildPhase, Language
    - Implementar métodos auxiliares para tipos
    - _Requisitos: 1.2, 7.1, 7.3_
  
  - [x] 2.2 Implementar carregamento de configuração com Viper
    - Criar struct Config com tags mapstructure
    - Implementar LoadConfig() com suporte a YAML e variáveis de ambiente
    - Implementar validateConfig() para validação de campos obrigatórios
    - _Requisitos: 10.1, 10.2_
  
  - [x] 2.3 Escrever testes unitários para shared library
    - Testar parsing de configuração YAML
    - Testar override com variáveis de ambiente
    - Testar validação de configuração inválida
    - Testar criação e manipulação de BuildJob
    - _Requisitos: 10.1, 10.2_

- [x] 3. Implementar NATS Client (libs/nats-client)
  - [x] 3.1 Criar wrapper do cliente NATS
    - Implementar interface NATSClient
    - Implementar Connect() com retry automático
    - Implementar Publish(), Subscribe(), Request()
    - Implementar Close() com graceful shutdown
    - Adicionar logging com Zap
    - _Requisitos: 1.5, 9.1_
  
  - [x] 3.2 Escrever testes unitários para NATS client
    - Testar conexão e reconexão
    - Testar publish/subscribe
    - Testar request/reply pattern
    - Usar NATS test server para testes
    - _Requisitos: 1.5_

- [ ] 4. Implementar Cache Service (libs/cache-service)
  - [x] 4.1 Criar gerenciador de cache
    - Implementar interface CacheService
    - Implementar GetCachePath() para cada linguagem
    - Implementar InitializeCache() para criar estrutura de diretórios
    - Implementar CleanCache() para limpeza de cache antigo
    - Implementar GetCacheSize() para monitoramento
    - _Requisitos: 4.1, 4.4_
  
  - [x] 4.2 Escrever testes unitários para cache service
    - Testar criação de estrutura de diretórios
    - Testar cálculo de tamanho de cache
    - Testar limpeza de arquivos antigos
    - Testar paths para diferentes linguagens
    - _Requisitos: 4.1, 4.4_
  
  - [x] 4.3 Escrever teste de propriedade para cache service
    - **Propriedade 10: Persistência de dependências em cache**
    - **Valida: Requisitos 4.3, 4.5**

- [ ] 5. Implementar Git Service (libs/git-service)
  - [x] 5.1 Criar gerenciador de operações Git
    - Implementar interface GitService
    - Implementar SyncRepository() com lógica de clone vs pull
    - Implementar RepositoryExists() para verificar cache
    - Implementar GetLocalPath() para calcular path local
    - Adicionar retry com backoff exponencial
    - Adicionar logging detalhado
    - _Requisitos: 2.1, 2.2, 2.3, 9.1_
  
  - [x] 5.2 Escrever testes unitários para git service
    - Testar clone de repositório novo
    - Testar pull de repositório existente
    - Testar detecção de repositório em cache
    - Testar cálculo de path local
    - Mockar operações go-git
    - _Requisitos: 2.1, 2.2, 2.3_
  
  - [x] 5.3 Escrever teste de propriedade para sincronização
    - **Propriedade 4: Sincronização de repositório**
    - **Valida: Requisitos 2.1, 2.2, 2.3**
  
  - [x] 5.4 Escrever teste de propriedade para fallback em falha de rede
    - **Propriedade 5: Fallback para cache em falha de rede**
    - **Valida: Requisitos 2.4**

- [x] 6. Implementar NX Service (libs/nx-service)
  - [x] 6.1 Criar executor de builds NX
    - Implementar interface NXService
    - Implementar Build() com execução de comando nx
    - Implementar DetectProjects() para descobrir projetos
    - Implementar detecção de linguagem baseada em arquivos
    - Configurar variáveis de ambiente para cache
    - Capturar stdout e stderr
    - Implementar timeout configurável
    - _Requisitos: 3.1, 3.2, 3.5, 6.4_
  
  - [x] 6.2 Escrever testes unitários para NX service
    - Testar execução de build bem-sucedido
    - Testar captura de output
    - Testar detecção de linguagem (Java, .NET, Go)
    - Testar configuração de cache
    - Testar timeout de build
    - Mockar execução de comandos
    - _Requisitos: 3.1, 3.2, 6.4_
  
  - [ ]* 6.3 Escrever teste de propriedade para captura de saída
    - **Propriedade 6: Captura de saída de build**
    - **Valida: Requisitos 3.2**
  
  - [ ]* 6.4 Escrever teste de propriedade para detecção de linguagem
    - **Propriedade 11: Detecção automática de linguagem**
    - **Valida: Requisitos 6.4**
  
  - [ ]* 6.5 Escrever teste de propriedade para configuração de cache
    - **Propriedade 9: Configuração de cache por linguagem**
    - **Valida: Requisitos 4.2, 6.1, 6.2, 6.3**

- [ ] 7. Implementar Image Service (libs/image-service)
  - [x] 7.1 Criar construtor de imagens OCI
    - Implementar interface ImageService
    - Implementar BuildImage() com execução de buildah
    - Implementar TagImage() para aplicar tags
    - Implementar localização de Dockerfile
    - Implementar validação de Dockerfile
    - Aplicar tags baseadas em commit hash e branch
    - _Requisitos: 5.1, 5.2, 5.4_
  
  - [ ]* 7.2 Escrever testes unitários para image service
    - Testar build de imagem
    - Testar aplicação de tags
    - Testar localização de Dockerfile
    - Testar validação de Dockerfile
    - Testar erro quando Dockerfile não existe
    - Mockar comandos buildah
    - _Requisitos: 5.1, 5.2, 5.3, 5.4_
  
  - [ ]* 7.3 Escrever teste de propriedade para localização de Dockerfile
    - **Propriedade 13: Localização de Dockerfile**
    - **Valida: Requisitos 5.2**
  
  - [ ]* 7.4 Escrever teste de propriedade para aplicação de tags
    - **Propriedade 15: Aplicação de tags de imagem**
    - **Valida: Requisitos 5.4**

- [ ] 8. Checkpoint - Verificar bibliotecas base
  - Executar testes unitários de todas as libs
  - Verificar cobertura de código (meta: 80%)
  - Garantir que todas as interfaces estão bem definidas
  - Perguntar ao usuário se há dúvidas ou ajustes necessários

- [ ] 9. Implementar API Service (apps/api-service)
  - [ ] 9.1 Configurar aplicação com FX e Gin
    - Criar main.go com FX app
    - Configurar providers FX (logger, config, nats client, gin router)
    - Implementar lifecycle hooks (OnStart, OnStop)
    - Carregar configuração com Viper
    - Criar config.yaml com valores padrão
    - _Requisitos: 10.1, 10.2_
  
  - [ ] 9.2 Implementar webhook handler
    - Criar WebhookHandler struct
    - Implementar validação de assinatura HMAC-SHA256
    - Implementar parsing de payload GitHub
    - Implementar extração de informações (repo, commit, branch)
    - Publicar BuildJob no NATS subject builds.webhook
    - Retornar HTTP 202 Accepted com job ID
    - _Requisitos: 1.1, 1.2, 1.4, 10.4_
  
  - [ ] 9.3 Implementar status handler
    - Criar StatusHandler struct
    - Implementar endpoint GET /builds/:id
    - Implementar endpoint GET /builds
    - Usar NATS request/reply para consultar status
    - Retornar JSON com informações do build
    - _Requisitos: 8.1, 8.2, 8.3_
  
  - [ ] 9.4 Implementar middleware de autenticação
    - Criar middleware para validar token de autenticação
    - Retornar HTTP 401 para requisições não autenticadas
    - Aplicar middleware em endpoints de consulta
    - _Requisitos: 8.4_
  
  - [ ] 9.5 Implementar middleware de logging
    - Criar middleware para logging de requisições
    - Registrar método, path, status code, duração
    - Usar Zap para logging estruturado
    - _Requisitos: 7.1_
  
  - [ ] 9.6 Implementar health check endpoint
    - Criar endpoint GET /health
    - Verificar conectividade com NATS
    - Retornar status do serviço
    - _Requisitos: 8.1_
  
  - [ ]* 9.7 Escrever testes unitários para API service
    - Testar webhook handler com payload válido
    - Testar webhook handler com assinatura inválida
    - Testar status handler
    - Testar middleware de autenticação
    - Testar health check
    - Mockar NATS client
    - _Requisitos: 1.1, 1.2, 1.3, 8.4_
  
  - [ ]* 9.8 Escrever teste de propriedade para validação de webhook
    - **Propriedade 1: Validação de assinatura de webhook**
    - **Valida: Requisitos 1.1, 10.4**
  
  - [ ]* 9.9 Escrever teste de propriedade para extração de informações
    - **Propriedade 2: Extração completa de informações de webhook**
    - **Valida: Requisitos 1.2**
  
  - [ ]* 9.10 Escrever teste de propriedade para autenticação
    - **Propriedade 23: Autenticação em endpoints de consulta**
    - **Valida: Requisitos 8.4**

- [ ] 10. Implementar Worker Service (apps/worker-service)
  - [ ] 10.1 Configurar aplicação com FX
    - Criar main.go com FX app
    - Configurar providers FX (logger, config, nats client, services)
    - Implementar lifecycle hooks
    - Carregar configuração com Viper
    - Criar config.yaml com valores padrão
    - _Requisitos: 10.1, 10.2_
  
  - [ ] 10.2 Implementar subscriber NATS
    - Criar subscriber para subject builds.webhook
    - Implementar pool de workers (goroutines)
    - Implementar fila de jobs usando channels
    - Implementar graceful shutdown
    - _Requisitos: 1.5_
  
  - [ ] 10.3 Implementar Build Orchestrator
    - Criar BuildOrchestrator struct
    - Implementar ExecuteBuild() com coordenação de fases
    - Implementar fase Git Sync (chamar GitService)
    - Implementar fase NX Build (chamar NXService)
    - Implementar fase Image Build (chamar ImageService)
    - Implementar context com timeout
    - Publicar status no NATS durante execução
    - Publicar conclusão no NATS ao finalizar
    - _Requisitos: 3.1, 3.3, 3.4, 5.1, 7.3_
  
  - [ ] 10.4 Implementar tratamento de erros
    - Implementar retry com backoff exponencial
    - Implementar rollback em caso de falha
    - Preservar logs e estado em falhas
    - Implementar timeout para operações longas
    - _Requisitos: 9.1, 9.2, 9.4_
  
  - [ ] 10.5 Implementar logging de métricas
    - Registrar início e fim de cada job
    - Registrar duração de cada fase
    - Registrar informações de commit
    - Usar Zap para logging estruturado em JSON
    - _Requisitos: 7.1, 7.2, 7.3, 7.4, 7.5_
  
  - [ ]* 10.6 Escrever testes unitários para worker service
    - Testar processamento de job bem-sucedido
    - Testar processamento de job com falha
    - Testar coordenação de fases
    - Testar timeout de build
    - Testar retry em falhas temporárias
    - Mockar todos os services
    - _Requisitos: 3.3, 3.4, 9.1, 9.4_
  
  - [ ]* 10.7 Escrever teste de propriedade para enfileiramento
    - **Propriedade 3: Enfileiramento de webhooks simultâneos**
    - **Valida: Requisitos 1.5**
  
  - [ ]* 10.8 Escrever teste de propriedade para interrupção em falha
    - **Propriedade 7: Interrupção em falha de build**
    - **Valida: Requisitos 3.3**
  
  - [ ]* 10.9 Escrever teste de propriedade para progressão após sucesso
    - **Propriedade 8: Progressão após build bem-sucedido**
    - **Valida: Requisitos 3.4**
  
  - [ ]* 10.10 Escrever teste de propriedade para logging
    - **Propriedade 17: Logging de início e fim de job**
    - **Valida: Requisitos 7.1**
  
  - [ ]* 10.11 Escrever teste de propriedade para métricas
    - **Propriedade 19: Métricas de duração por fase**
    - **Valida: Requisitos 7.3**

- [ ] 11. Checkpoint - Verificar serviços principais
  - Executar testes unitários de api-service e worker-service
  - Testar comunicação via NATS localmente
  - Verificar logs estruturados
  - Perguntar ao usuário se há dúvidas ou ajustes necessários

- [ ] 12. Criar Dockerfiles para serviços
  - [ ] 12.1 Criar Dockerfile para api-service
    - Multi-stage build (build + runtime)
    - Copiar binário compilado
    - Expor porta 8080
    - Definir entrypoint
    - _Requisitos: 10.1_
  
  - [ ] 12.2 Criar Dockerfile para worker-service
    - Multi-stage build (build + runtime)
    - Instalar buildah no container
    - Copiar binário compilado
    - Montar socket do Docker
    - Definir entrypoint
    - _Requisitos: 10.1_

- [ ] 13. Configurar NX para build dos serviços
  - [ ] 13.1 Criar project.json para api-service
    - Configurar target build
    - Configurar target test
    - Configurar target serve
    - Configurar dependências
    - _Requisitos: 10.1_
  
  - [ ] 13.2 Criar project.json para worker-service
    - Configurar target build
    - Configurar target test
    - Configurar dependências
    - _Requisitos: 10.1_
  
  - [ ] 13.3 Criar project.json para cada lib
    - Configurar targets para git-service, nx-service, image-service, cache-service, nats-client, shared
    - Configurar dependências entre libs
    - _Requisitos: 10.1_
  
  - [ ] 13.4 Testar builds com NX
    - Executar nx build api-service
    - Executar nx build worker-service
    - Executar nx affected:build
    - Verificar cache do NX
    - _Requisitos: 10.1_

- [ ] 14. Atualizar docker-compose.yml
  - Adicionar volumes para cache e logs
  - Configurar variáveis de ambiente
  - Configurar dependências entre serviços
  - Configurar réplicas do worker-service
  - Adicionar health checks
  - _Requisitos: 10.1_

- [ ] 15. Criar testes de integração com Robot Framework
  - [ ] 15.1 Configurar ambiente de testes Robot
    - Instalar Robot Framework e bibliotecas (RequestsLibrary)
    - Criar estrutura de diretórios (tests/integration)
    - Criar keywords customizadas
    - Criar variáveis de teste
    - _Requisitos: 8.1, 8.2_
  
  - [ ] 15.2 Criar repositórios de teste
    - Criar sample-java-repo com pom.xml e Dockerfile
    - Criar sample-dotnet-repo com .csproj e Dockerfile
    - Criar sample-go-repo com go.mod e Dockerfile
    - Adicionar em tests/integration/fixtures
    - _Requisitos: 6.1, 6.2, 6.3_
  
  - [ ] 15.3 Criar webhook.robot
    - Teste: enviar webhook válido e verificar enfileiramento
    - Teste: enviar webhook com assinatura inválida
    - Teste: enviar múltiplos webhooks simultâneos
    - Teste: verificar parsing de payload
    - _Requisitos: 1.1, 1.2, 1.3, 1.5_
  
  - [ ] 15.4 Criar build.robot
    - Teste: build completo de projeto Java
    - Teste: build completo de projeto .NET
    - Teste: build completo de projeto Go
    - Teste: build com falha (código não compila)
    - Teste: build com Dockerfile ausente
    - Teste: build com cache de dependências
    - _Requisitos: 3.1, 3.3, 5.3, 6.1, 6.2, 6.3_
  
  - [ ] 15.5 Criar api.robot
    - Teste: consultar status de build existente
    - Teste: consultar status de build inexistente
    - Teste: listar histórico de builds
    - Teste: health check
    - Teste: autenticação com token válido
    - Teste: autenticação com token inválido
    - _Requisitos: 8.1, 8.2, 8.3, 8.4, 8.5_

- [ ] 16. Checkpoint final - Testes end-to-end
  - Subir ambiente completo com docker-compose
  - Executar todos os testes Robot Framework
  - Verificar logs de todos os serviços
  - Verificar imagens OCI criadas
  - Verificar cache de código e dependências
  - Perguntar ao usuário se há dúvidas ou ajustes necessários

- [ ] 17. Documentação e finalização
  - [ ] 17.1 Criar README.md
    - Documentar arquitetura do sistema
    - Documentar como executar localmente
    - Documentar como executar testes
    - Documentar configuração de webhooks GitHub
    - Documentar variáveis de ambiente
    - _Requisitos: 10.1_
  
  - [ ] 17.2 Criar guia de desenvolvimento
    - Documentar estrutura do monorepo
    - Documentar como adicionar novos serviços
    - Documentar como adicionar novos testes
    - Documentar comandos NX úteis
    - _Requisitos: 10.1_
  
  - [ ] 17.3 Criar guia de deployment
    - Documentar requisitos de infraestrutura
    - Documentar como fazer deploy dos serviços
    - Documentar configuração de NATS em produção
    - Documentar monitoramento e observabilidade
    - _Requisitos: 10.1_

## Notas

- Tarefas marcadas com `*` são opcionais e podem ser puladas para um MVP mais rápido
- Cada tarefa referencia requisitos específicos para rastreabilidade
- Checkpoints garantem validação incremental
- Testes de propriedade validam propriedades universais de corretude
- Testes unitários validam exemplos específicos e casos extremos
- Testes de integração Robot Framework validam fluxos end-to-end
