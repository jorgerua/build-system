# Guia de Build e Desenvolvimento

Este documento descreve como usar os comandos de build, teste e execução do OCI Build System.

## Pré-requisitos

- **Node.js** (v18 ou superior)
- **Go** (v1.21 ou superior)
- **Docker** e **Docker Compose** (para executar serviços)
- **Make** (Linux/Mac) ou **PowerShell** (Windows)

## Comandos Disponíveis

### Linux/Mac (usando Makefile)

```bash
# Mostrar ajuda
make help

# Instalar dependências
make install

# Build
make build              # Build de todos os projetos
make build-affected     # Build apenas dos projetos afetados
make build-shared       # Build da biblioteca shared
make build-api          # Build do api-service
make build-worker       # Build do worker-service

# Testes
make test               # Testes de todos os projetos
make test-affected      # Testes dos projetos afetados
make test-shared        # Testes da biblioteca shared
make test-coverage      # Testes com cobertura de código

# Execução
make run                # Inicia todos os serviços (docker-compose)
make run-api            # Executa api-service localmente
make run-worker         # Executa worker-service localmente
make run-nats           # Inicia apenas o NATS
make stop               # Para todos os serviços
make restart            # Reinicia todos os serviços

# Logs
make logs               # Logs de todos os serviços
make logs-api           # Logs do api-service
make logs-worker        # Logs do worker-service
make logs-nats          # Logs do NATS

# Desenvolvimento
make dev                # Inicia ambiente de desenvolvimento
make format             # Formata código Go
make lint               # Executa linting
make clean              # Remove arquivos de build e cache
make graph              # Mostra gráfico de dependências NX

# CI/CD
make ci                 # Executa pipeline completo de CI
```

### Windows (usando PowerShell)

```powershell
# Mostrar ajuda
.\build.ps1 help

# Instalar dependências
.\build.ps1 install

# Build
.\build.ps1 build              # Build de todos os projetos
.\build.ps1 build-affected     # Build apenas dos projetos afetados
.\build.ps1 build-shared       # Build da biblioteca shared

# Testes
.\build.ps1 test               # Testes de todos os projetos
.\build.ps1 test-affected      # Testes dos projetos afetados
.\build.ps1 test-shared        # Testes da biblioteca shared
.\build.ps1 test-coverage      # Testes com cobertura de código

# Execução
.\build.ps1 run                # Inicia todos os serviços (docker-compose)
.\build.ps1 run-nats           # Inicia apenas o NATS
.\build.ps1 stop               # Para todos os serviços

# Desenvolvimento
.\build.ps1 format             # Formata código Go
.\build.ps1 clean              # Remove arquivos de build e cache
.\build.ps1 graph              # Mostra gráfico de dependências NX

# CI/CD
.\build.ps1 ci                 # Executa pipeline completo de CI
```

## Usando NX Diretamente

Você também pode usar o NX diretamente para comandos mais específicos:

```bash
# Build
npm exec nx run shared:build
npm exec nx run api-service:build
npm exec nx run-many -- --target=build --all

# Testes
npm exec nx run shared:test
npm exec nx run-many -- --target=test --all

# Build apenas do que foi afetado
npm exec nx affected -- --target=build

# Visualizar gráfico de dependências
npm exec nx graph

# Limpar cache do NX
npm exec nx reset
```

## Fluxo de Desenvolvimento Típico

### 1. Configuração Inicial

```bash
# Linux/Mac
make install

# Windows
.\build.ps1 install
```

### 2. Desenvolvimento Local

```bash
# Inicia NATS em background
make run-nats  # ou .\build.ps1 run-nats

# Em um terminal, execute o api-service
make run-api

# Em outro terminal, execute o worker-service
make run-worker
```

### 3. Executar Testes

```bash
# Testes de um projeto específico
make test-shared  # ou .\build.ps1 test-shared

# Testes de todos os projetos
make test  # ou .\build.ps1 test

# Testes apenas do que foi afetado
make test-affected  # ou .\build.ps1 test-affected
```

### 4. Build para Produção

```bash
# Build de todos os projetos
make build  # ou .\build.ps1 build

# Build apenas do que foi afetado
make build-affected  # ou .\build.ps1 build-affected
```

### 5. Executar com Docker Compose

```bash
# Inicia todos os serviços
make run  # ou .\build.ps1 run

# Ver logs
make logs  # ou docker-compose logs -f

# Parar serviços
make stop  # ou .\build.ps1 stop
```

## Estrutura de Projetos NX

Cada projeto (libs e apps) possui um arquivo `project.json` que define:

- **build**: Compila o código Go
- **test**: Executa testes unitários
- **lint**: Executa verificação de código

O NX gerencia automaticamente:
- Cache de builds e testes
- Dependências entre projetos
- Execução paralela de tarefas
- Detecção de projetos afetados por mudanças

## Cache do NX

O NX mantém cache de builds e testes para acelerar execuções subsequentes:

- Cache local em `.nx/cache`
- Reutiliza resultados quando código não mudou
- Use `make clean` ou `npm exec nx reset` para limpar o cache

## Troubleshooting

### Build falha com erro de dependências Go

```bash
cd libs/shared
go mod tidy
cd ../..
make build-shared
```

### Cache do NX causando problemas

```bash
make clean  # ou .\build.ps1 clean
npm exec nx reset
```

### Docker Compose não inicia

```bash
# Verificar se portas estão em uso
docker-compose ps

# Parar e remover containers
docker-compose down -v

# Iniciar novamente
make run
```

### Permissões no Linux/Mac

```bash
# Dar permissão de execução ao Makefile
chmod +x Makefile
```

## Integração Contínua (CI)

Para executar o pipeline completo de CI:

```bash
make ci  # ou .\build.ps1 ci
```

Isso irá:
1. Verificar dependências
2. Instalar pacotes
3. Executar build de todos os projetos
4. Executar todos os testes

## Recursos Adicionais

- [Documentação do NX](https://nx.dev)
- [Documentação do Go](https://go.dev/doc/)
- [Docker Compose](https://docs.docker.com/compose/)
