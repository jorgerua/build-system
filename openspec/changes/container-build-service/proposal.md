## Why

Precisamos de um serviço centralizado que automatize o build de container images a partir de monorepos GitHub baseados em Nx. Hoje não existe um sistema que entenda a estrutura do monorepo, detecte quais projetos foram afetados por um push, e construa apenas os containers necessários com versionamento semântico automático.

## What Changes

- Novo serviço Go composto por **webhook receiver** (HTTP) e **worker** (consumer NATS) deployados em Kubernetes.
- Webhook receiver valida push events do GitHub (autenticação via GitHub App) e publica jobs no NATS.
- Worker clona o repositório, executa `nx affected` para detectar projetos impactados, e executa builds Buildah em paralelo diretamente no pod do worker (sem pods efêmeros).
- Dockerfiles são gerados automaticamente pelo serviço com base na linguagem detectada (Go, Java, .NET) — Dockerfiles existentes no repo são ignorados.
- Versionamento SemVer 2.0.0 por projeto, derivado de Conventional Commits (default: patch). Estado de versões e último SHA processado persistidos em TiDB.
- Cache Nx em PVC compartilhado e buildah storage em PVC por worker para máxima performance de build.
- Retry automático de builds falhos (máximo 3 tentativas).
- Observabilidade com logs estruturados (zap/JSON) e métricas de build via Datadog (DogStatsD).

## Capabilities

### New Capabilities
- `webhook-receiver`: Recepção e validação de push webhooks do GitHub, autenticação via GitHub App, publicação de jobs no NATS.
- `build-orchestrator`: Worker que consome jobs, clona repos, executa nx affected, detecta projetos buildable por convenção de diretório (apps/*), e orquestra builds paralelos.
- `language-detection`: Detecção automática de linguagem do projeto (Go via go.mod, Java via pom.xml, .NET via .csproj) para seleção de template de Dockerfile.
- `dockerfile-templates`: Templates padronizados de Dockerfile multi-stage para Go, Java e .NET.
- `semver-versioning`: Cálculo de versão SemVer baseado em Conventional Commits, com persistência por projeto em TiDB.
- `container-builder`: Execução de builds de container via `buildah bud` no worker pod, usando o clone local como build context e push direto para o registry.
- `build-cache`: Gerenciamento de cache Nx via PVC compartilhado (RWX) e buildah storage via PVC por worker (RWO).
- `build-observability`: Logs estruturados via zap e métricas de build (duração, sucesso/falha, retry count) via Datadog.

### Modified Capabilities

## Impact

- **Infraestrutura K8s**: Novos deployments (webhook-server, worker), PVCs para nx-cache (RWX) e buildah-storage (RWO por worker), SecurityContext com capabilities para buildah.
- **NATS**: Novo subject/stream para jobs de build.
- **TiDB**: Novas tabelas para versões de projetos e SHA processados.
- **GitHub**: Configuração de GitHub App com permissões de leitura no repositório e webhook de push.
- **Container Registry**: Push de imagens com tags SemVer.
- **Datadog**: Novos dashboards e métricas de build.
