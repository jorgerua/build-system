# Tasks: Melhoria de Testes Unitários e Integrados

## Overview

Este documento contém as tarefas para implementar melhorias nos testes unitários e integrados do OCI Build System, incluindo correção de bugs de configuração, aumento de cobertura para ~80%, e automação via Makefile.

## Task List

### Phase 1: Configuration Fixes (High Priority)

- [x] 1. Criar arquivo .env.example
  - [x] 1.1 Documentar todas as variáveis de ambiente necessárias
  - [x] 1.2 Adicionar comentários explicativos para cada variável
  - [x] 1.3 Incluir valores de exemplo para desenvolvimento

- [x] 2. Atualizar docker-compose.yml com validação de variáveis
  - [x] 2.1 Adicionar sintaxe ${VAR:?message} para variáveis obrigatórias
  - [x] 2.2 Adicionar health checks para NATS
  - [x] 2.3 Adicionar health checks para api-service
  - [x] 2.4 Configurar depends_on com condition: service_healthy
  - [x] 2.5 Testar que compose falha se variáveis obrigatórias não estão definidas

- [ ] 3. Implementar função LoadConfig com validação
  - [ ] 3.1 Criar função expandEnvVars para expandir ${VAR_NAME}
  - [ ] 3.2 Criar função validateConfig com validação de campos obrigatórios
  - [ ] 3.3 Criar função validateCachePath para verificar paths
  - [ ] 3.4 Criar função logConfig para log de configuração (sem secrets)
  - [ ] 3.5 Atualizar LoadConfig para usar as novas funções

- [ ] 4. Adicionar testes unitários para configuração
  - [ ] 4.1 Testar LoadConfig com arquivo válido
  - [ ] 4.2 Testar LoadConfig com variáveis de ambiente
  - [ ] 4.3 Testar validateConfig com configuração inválida
  - [ ] 4.4 Testar expandEnvVars com diferentes formatos
  - [ ] 4.5 Testar validateCachePath com paths válidos e inválidos

### Phase 2: Health Checks (High Priority)

- [ ] 5. Implementar endpoints de health check
  - [ ] 5.1 Atualizar HealthHandler com verificação de NATS
  - [ ] 5.2 Implementar endpoint /health com status detalhado
  - [ ] 5.3 Implementar endpoint /readiness
  - [ ] 5.4 Implementar endpoint /liveness
  - [ ] 5.5 Registrar endpoints no router

- [ ] 6. Adicionar testes para health handlers
  - [ ] 6.1 Testar /health quando NATS está conectado
  - [ ] 6.2 Testar /health quando NATS está desconectado
  - [ ] 6.3 Testar /readiness quando serviço está pronto
  - [ ] 6.4 Testar /readiness quando serviço não está pronto
  - [ ] 6.5 Testar /liveness sempre retorna 200

- [ ] 7. Criar script wait-for-services.sh
  - [ ] 7.1 Implementar função check_service
  - [ ] 7.2 Adicionar verificação de NATS
  - [ ] 7.3 Adicionar verificação de API Service
  - [ ] 7.4 Adicionar retry com backoff
  - [ ] 7.5 Adicionar output colorido e informativo
  - [ ] 7.6 Tornar script executável (chmod +x)

### Phase 3: Test Utilities (Medium Priority)

- [ ] 8. Criar package tests/testutil
  - [ ] 8.1 Criar diretório tests/testutil
  - [ ] 8.2 Criar arquivo mocks.go com MockNATSClient
  - [ ] 8.3 Adicionar MockGitService
  - [ ] 8.4 Adicionar MockNXService
  - [ ] 8.5 Adicionar MockImageService
  - [ ] 8.6 Adicionar MockCacheService

- [ ] 9. Implementar fixtures helpers
  - [ ] 9.1 Criar arquivo fixtures.go
  - [ ] 9.2 Implementar CreateTempRepo para diferentes linguagens
  - [ ] 9.3 Implementar LoadWebhookPayload
  - [ ] 9.4 Implementar GenerateHMACSignature
  - [ ] 9.5 Adicionar constantes para fixtures (javaPomXML, dotnetCsproj, etc)

- [ ] 10. Implementar assertion helpers
  - [ ] 10.1 Criar arquivo assertions.go
  - [ ] 10.2 Implementar AssertBuildJobValid
  - [ ] 10.3 Implementar AssertHTTPStatus
  - [ ] 10.4 Implementar AssertJSONResponse
  - [ ] 10.5 Adicionar outras assertions úteis

### Phase 4: Unit Test Coverage (High Priority)

- [ ] 11. Analisar cobertura atual
  - [ ] 11.1 Executar go test com coverage
  - [ ] 11.2 Gerar relatório HTML
  - [ ] 11.3 Identificar componentes com < 75% cobertura
  - [ ] 11.4 Priorizar componentes críticos

- [ ] 12. Adicionar testes para API Service handlers
  - [ ] 12.1 Adicionar testes faltantes para webhook handler
  - [ ] 12.2 Adicionar testes faltantes para status handler
  - [ ] 12.3 Adicionar testes para casos de erro
  - [ ] 12.4 Adicionar testes para edge cases

- [ ] 13. Adicionar testes para middlewares
  - [ ] 13.1 Criar logging_test.go
  - [ ] 13.2 Testar LoggingMiddleware com request normal
  - [ ] 13.3 Testar LoggingMiddleware com erro
  - [ ] 13.4 Testar LoggingMiddleware inclui latência
  - [ ] 13.5 Adicionar testes faltantes para auth middleware

- [ ] 14. Adicionar testes para NX Service
  - [ ] 14.1 Criar builder_test.go se não existir
  - [ ] 14.2 Testar DetectLanguage para Java
  - [ ] 14.3 Testar DetectLanguage para .NET
  - [ ] 14.4 Testar DetectLanguage para Go
  - [ ] 14.5 Testar DetectLanguage para linguagem desconhecida
  - [ ] 14.6 Testar Build com timeout
  - [ ] 14.7 Testar Build com erro de execução

- [ ] 15. Adicionar testes para Git Service
  - [ ] 15.1 Adicionar testes faltantes para SyncRepository
  - [ ] 15.2 Testar clone de repositório novo
  - [ ] 15.3 Testar pull de repositório existente
  - [ ] 15.4 Testar retry em falha de rede
  - [ ] 15.5 Testar fallback para cache

- [ ] 16. Adicionar testes para Image Service
  - [ ] 16.1 Adicionar testes faltantes para BuildImage
  - [ ] 16.2 Testar TagImage
  - [ ] 16.3 Testar validação de Dockerfile
  - [ ] 16.4 Testar erro quando Dockerfile não existe

- [ ] 17. Adicionar testes para Cache Service
  - [ ] 17.1 Verificar cobertura atual de manager_test.go
  - [ ] 17.2 Adicionar testes faltantes se necessário
  - [ ] 17.3 Testar CleanCache com diferentes cenários
  - [ ] 17.4 Testar GetCacheSize para todas as linguagens

- [ ] 18. Adicionar testes para Worker Service
  - [ ] 18.1 Adicionar testes faltantes para orchestrator
  - [ ] 18.2 Testar processamento de build completo
  - [ ] 18.3 Testar falha em cada fase
  - [ ] 18.4 Testar retry com backoff

- [ ] 19. Adicionar testes para NATS Client
  - [ ] 19.1 Criar client_test.go se não existir
  - [ ] 19.2 Testar Connect
  - [ ] 19.3 Testar Publish
  - [ ] 19.4 Testar Subscribe
  - [ ] 19.5 Testar Request/Reply
  - [ ] 19.6 Testar reconexão automática

- [ ] 20. Verificar meta de cobertura
  - [ ] 20.1 Executar testes com coverage
  - [ ] 20.2 Verificar que cobertura >= 75%
  - [ ] 20.3 Gerar relatório final
  - [ ] 20.4 Documentar componentes que não atingiram meta

### Phase 5: Property-Based Tests (Medium Priority)

- [ ] 21. Implementar property test para validação de assinatura
  - [ ] 21.1 Criar TestProperty_WebhookSignatureValidation
  - [ ] 21.2 Gerar payloads e secrets aleatórios
  - [ ] 21.3 Verificar que assinatura inválida retorna 401
  - [ ] 21.4 Executar 100+ iterações

- [ ] 22. Implementar property test para extração de webhook
  - [ ] 22.1 Criar TestProperty_WebhookInformationExtraction
  - [ ] 22.2 Gerar webhooks válidos aleatórios
  - [ ] 22.3 Verificar que todas as informações são extraídas
  - [ ] 22.4 Executar 100+ iterações

- [ ] 23. Implementar property test para detecção de linguagem
  - [ ] 23.1 Criar TestProperty_LanguageDetection
  - [ ] 23.2 Gerar repositórios com diferentes arquivos de config
  - [ ] 23.3 Verificar que linguagem correta é detectada
  - [ ] 23.4 Executar 100+ iterações

- [ ] 24. Implementar property test para tags de imagem
  - [ ] 24.1 Criar TestProperty_ImageTagging
  - [ ] 24.2 Gerar commit hashes e branches aleatórios
  - [ ] 24.3 Verificar que ambas as tags são aplicadas
  - [ ] 24.4 Executar 100+ iterações

### Phase 6: Makefile Integration (High Priority)

- [ ] 25. Adicionar comandos de teste unitário ao Makefile
  - [ ] 25.1 Adicionar target test-unit com cobertura
  - [ ] 25.2 Adicionar target test-unit-quick sem cobertura
  - [ ] 25.3 Adicionar target test-property
  - [ ] 25.4 Adicionar target test-coverage-report
  - [ ] 25.5 Testar todos os comandos

- [ ] 26. Adicionar comandos de teste integrado ao Makefile
  - [ ] 26.1 Adicionar target test-integration-setup
  - [ ] 26.2 Adicionar target test-integration-teardown
  - [ ] 26.3 Adicionar target test-integration
  - [ ] 26.4 Adicionar target test-integration-keep
  - [ ] 26.5 Adicionar target test-integration-logs
  - [ ] 26.6 Testar todos os comandos

- [ ] 27. Adicionar comandos auxiliares ao Makefile
  - [ ] 27.1 Adicionar target test-all
  - [ ] 27.2 Adicionar target test-quick
  - [ ] 27.3 Adicionar target validate-env
  - [ ] 27.4 Adicionar target clean-test-reports
  - [ ] 27.5 Atualizar target help com novos comandos

### Phase 7: Integration Test Improvements (Medium Priority)

- [ ] 28. Atualizar Robot Framework keywords
  - [ ] 28.1 Atualizar resources/keywords.robot
  - [ ] 28.2 Implementar Setup Test Environment
  - [ ] 28.3 Implementar Teardown Test Environment
  - [ ] 28.4 Implementar Send Webhook
  - [ ] 28.5 Implementar Get Build Status
  - [ ] 28.6 Implementar Wait Until Build Completes
  - [ ] 28.7 Implementar Generate Random Commit Hash
  - [ ] 28.8 Implementar Create Webhook Payload
  - [ ] 28.9 Implementar Generate HMAC Signature
  - [ ] 28.10 Implementar Verify Response Status
  - [ ] 28.11 Implementar Verify JSON Response

- [ ] 29. Atualizar Robot Framework variables
  - [ ] 29.1 Atualizar resources/variables.robot
  - [ ] 29.2 Definir variáveis de API
  - [ ] 29.3 Definir constantes de HTTP status
  - [ ] 29.4 Definir variáveis de repositórios de teste
  - [ ] 29.5 Definir timeouts

- [ ] 30. Melhorar fixtures de teste
  - [ ] 30.1 Verificar fixtures/sample-java-repo
  - [ ] 30.2 Verificar fixtures/sample-dotnet-repo
  - [ ] 30.3 Verificar fixtures/sample-go-repo
  - [ ] 30.4 Adicionar Dockerfiles se faltando
  - [ ] 30.5 Adicionar código fonte mínimo

- [ ] 31. Adicionar testes integrados faltantes
  - [ ] 31.1 Revisar webhook.robot
  - [ ] 31.2 Revisar build.robot
  - [ ] 31.3 Revisar api.robot
  - [ ] 31.4 Adicionar testes para cenários não cobertos

### Phase 8: Documentation (Medium Priority)

- [ ] 32. Criar TESTING.md
  - [ ] 32.1 Adicionar seção de Visão Geral
  - [ ] 32.2 Adicionar seção de Meta de Cobertura
  - [ ] 32.3 Adicionar seção de Executando Testes
  - [ ] 32.4 Adicionar seção de Escrevendo Testes
  - [ ] 32.5 Adicionar seção de Debugging Testes
  - [ ] 32.6 Adicionar seção de Ambiente de Teste
  - [ ] 32.7 Adicionar seção de Troubleshooting
  - [ ] 32.8 Adicionar seção de Contribuindo

- [ ] 33. Adicionar exemplos de código
  - [ ] 33.1 Adicionar exemplo de teste unitário
  - [ ] 33.2 Adicionar exemplo de table-driven test
  - [ ] 33.3 Adicionar exemplo de property-based test
  - [ ] 33.4 Adicionar exemplo de teste integrado Robot

- [ ] 34. Documentar processo de validação de testes
  - [ ] 34.1 Criar checklist de validação
  - [ ] 34.2 Documentar passos de validação
  - [ ] 34.3 Adicionar exemplos de debugging

### Phase 9: CI/CD Integration (Low Priority)

- [ ] 35. Configurar GitHub Actions para testes
  - [ ] 35.1 Criar .github/workflows/test.yml
  - [ ] 35.2 Adicionar job de testes unitários
  - [ ] 35.3 Adicionar job de testes integrados
  - [ ] 35.4 Configurar secrets necessários
  - [ ] 35.5 Testar workflow

- [ ] 36. Configurar upload de relatórios
  - [ ] 36.1 Configurar upload de coverage para codecov
  - [ ] 36.2 Configurar artifacts para relatórios
  - [ ] 36.3 Adicionar badge de cobertura ao README

### Phase 10: Validation and Refinement (High Priority)

- [ ] 37. Validação completa do ambiente
  - [ ] 37.1 Executar make test-unit localmente
  - [ ] 37.2 Verificar que cobertura >= 75%
  - [ ] 37.3 Executar make test-integration localmente
  - [ ] 37.4 Verificar que todos os testes integrados passam
  - [ ] 37.5 Verificar que docker-compose sobe sem erros

- [ ] 38. Correção de bugs encontrados
  - [ ] 38.1 Documentar bugs encontrados durante validação
  - [ ] 38.2 Priorizar bugs críticos
  - [ ] 38.3 Corrigir bugs um por um
  - [ ] 38.4 Re-executar testes após cada correção

- [ ] 39. Documentação final
  - [ ] 39.1 Atualizar README.md com instruções de teste
  - [ ] 39.2 Documentar problemas conhecidos
  - [ ] 39.3 Adicionar seção de troubleshooting ao README
  - [ ] 39.4 Revisar toda a documentação

- [ ] 40. Validação final
  - [ ] 40.1 Executar make test-all
  - [ ] 40.2 Verificar todos os critérios de sucesso
  - [ ] 40.3 Gerar relatórios finais
  - [ ] 40.4 Marcar spec como completa

## Notes

### Prioridades

- **High Priority**: Phases 1, 2, 4, 6, 10
- **Medium Priority**: Phases 3, 5, 7, 8
- **Low Priority**: Phase 9

### Dependências

- Phase 3 deve ser completada antes de Phase 4 (testes precisam de utilities)
- Phase 1 e 2 devem ser completadas antes de Phase 6 (Makefile depende de configuração)
- Phase 6 deve ser completada antes de Phase 10 (validação usa Makefile)

### Estimativas de Tempo

- Phase 1: 4-6 horas
- Phase 2: 3-4 horas
- Phase 3: 4-5 horas
- Phase 4: 8-12 horas (maior fase)
- Phase 5: 4-6 horas
- Phase 6: 2-3 horas
- Phase 7: 3-4 horas
- Phase 8: 4-5 horas
- Phase 9: 2-3 horas
- Phase 10: 4-6 horas

**Total Estimado**: 38-54 horas

### Success Criteria

- [ ] Cobertura de testes unitários >= 75%
- [ ] Todos os testes unitários passam
- [ ] Todos os testes integrados passam
- [ ] Ambiente Docker Compose sobe sem erros
- [ ] Variáveis de ambiente são validadas corretamente
- [ ] Health checks funcionam corretamente
- [ ] Testes podem ser executados via Makefile
- [ ] Relatórios de cobertura são gerados
- [ ] Documentação está completa
- [ ] Property-based tests executam >= 100 iterações
