## Context

Projeto greenfield — não existe serviço de build de containers atualmente. O monorepo alvo usa Nx como build system e hospeda múltiplos projetos em Go, Java e .NET sob a convenção `apps/<project>`. O cluster Kubernetes já existe e roda NATS e TiDB como infraestrutura compartilhada. Datadog já está presente como solução de observabilidade.

O serviço será composto por dois binários Go: um **webhook server** (HTTP) e um **worker** (consumer NATS), ambos usando a stack fx/zap/viper.

## Goals / Non-Goals

**Goals:**
- Buildar automaticamente containers dos projetos afetados por um push ao monorepo.
- Detectar projetos afetados via `nx affected`, limitado a projetos sob `apps/`.
- Gerar Dockerfiles padronizados por linguagem (Go, Java, .NET), ignorando quaisquer Dockerfiles do repo.
- Versionar cada projeto independentemente com SemVer, derivado de Conventional Commits.
- Executar builds em paralelo via `buildah bud` como subprocesso no worker pod (sem pods efêmeros).
- Maximizar performance de build via cache Nx em PVC compartilhado e cache de layers buildah em PVC por worker.

**Non-Goals:**
- Suporte a múltiplos monorepos (single-repo por deployment).
- Suporte a linguagens além de Go, Java e .NET nesta fase.
- Deploy automático das imagens (apenas build + push).
- Geração de Dockerfiles customizáveis por projeto.
- CI/CD genérico — o serviço faz exclusivamente build de containers.

## Decisions

### 1. Dois binários separados (webhook-server e worker)

**Escolha**: Separar webhook receiver e worker em binários distintos, ambos no mesmo módulo Go.

**Alternativas consideradas**:
- Binário único com flag de modo: simples, mas acoplaria scaling do receiver com scaling dos workers.
- Três binários (receiver + orchestrator + builder): overengineering para o escopo atual.

**Rationale**: Webhook server precisa ser leve e responder rápido (< 10s pro GitHub não fazer retry). Workers precisam escalar independentemente conforme a carga de builds. Mesmo módulo Go permite compartilhar packages internos.

### 2. NATS JetStream como fila de jobs

**Escolha**: Usar NATS JetStream com consumer durável para a fila de build jobs.

**Alternativas consideradas**:
- NATS Core (pub/sub simples): sem garantia de entrega.
- Redis + BullMQ: exigiria Node.js sidecar ou reimplementação do protocolo em Go.
- RabbitMQ: mais componente de infra; NATS já está no cluster.

**Rationale**: JetStream oferece at-least-once delivery, ack/nack, e retry nativo. Já está disponível no cluster. Consumer durável garante que jobs não se percam se workers reiniciarem.

### 3. `nx affected` com SHA do payload + último SHA processado do DB

**Escolha**: Worker usa `nx affected --base=<last_processed_sha> --head=<push_after_sha>`. O `last_processed_sha` é armazenado no TiDB e atualizado após processamento bem-sucedido.

**Alternativas consideradas**:
- Usar campo `before` do payload do webhook: falha se webhooks forem perdidos (gap entre builds).
- Sempre comparar com `HEAD~1`: não captura pushes com múltiplos commits.

**Rationale**: Guardar o último SHA processado no DB é resiliente a webhooks perdidos e garante que nenhum commit seja ignorado.

**Primeiro run (sem `last_processed_sha` no DB)**: O worker obtém o commit inicial do repositório via `git rev-list --max-parents=0 HEAD` no clone local e o usa como `--base`. `nx affected --base=<initial-commit-sha> --head=<push-after-sha>` marca todos os projetos existentes como afetados — reutilizando o mesmo code path do fluxo normal, sem branch especial de "primeiro run".

### 4. Detecção de linguagem por arquivo marcador

**Escolha**: Detectar linguagem do projeto pela presença de arquivos específicos na raiz do projeto:
- `go.mod` → Go
- `pom.xml` → Java (Maven)
- `build.gradle` ou `build.gradle.kts` → Java (Gradle)
- `*.csproj` → .NET

**Alternativas consideradas**:
- Configuração explícita no project.json do Nx: requer mudanças no monorepo.
- Análise de extensões de arquivos: ambíguo e frágil.

**Rationale**: Arquivos marcadores são determinísticos e já existem naturalmente nos projetos. Prioridade definida para resolver ambiguidades (ex: projeto com go.mod e package.json → Go).

### 5. Buildah executado no worker pod (sem pods efêmeros)

**Escolha**: Builds de container são executados pelo binário `buildah bud` como subprocesso dentro do próprio worker pod, usando o clone local do repositório já disponível no filesystem como build context. Não são criados pods adicionais para os builds.

**Alternativas consideradas**:
- Kaniko em pods efêmeros (design original): isolamento por build, mas com complexidade de ciclo de vida de pods, latência de scheduling, necessidade de RBAC para criação de pods, segundo clone do repositório e gestão de ConfigMaps para Dockerfiles.
- DinD (Docker-in-Docker): requer `privileged: true` no pod, considerado inseguro em ambientes multi-tenant.
- Podman: similar ao Buildah para builds; Buildah é preferido por ser focado exclusivamente em construção de imagens (sem daemon de runtime), resultando em imagem mais enxuta.

**Rationale**: Buildah elimina toda a complexidade de orquestração de pods — sem `client-go`, sem RBAC para criar pods, sem ConfigMaps, sem segundo clone do repositório. O build context é o clone local já presente no filesystem do worker, reutilizado da fase de análise (`nx affected`). O binário `buildah` usa o secret de registry montado no pod via `--authfile`.

**Consequências**:
- Worker image precisa do binário `buildah` instalado (multi-stage build: Go builder + buildah runtime)
- SecurityContext do pod precisa de `CAP_SETUID` e `CAP_SETFCAP` para buildah com overlay filesystem; alternativa rootless com `--storage-driver vfs` (sem capabilities adicionais, mais lento)
- O semáforo de concorrência controla processos `buildah bud` paralelos dentro do worker
- Clone local do worker é reutilizado como build context, eliminando o segundo clone
- Buildah storage (camadas cacheadas) persistida em PVC montado no worker

### 6. Cache Nx em PVC compartilhado

**Escolha**: Workers montam um PVC (ReadWriteMany) onde o Nx persiste cache de computação (`.nx-cache`).

**Rationale**: `nx affected` e tasks de build do Nx usam hash-based caching. PVC compartilhado entre workers permite reuso de cache entre execuções. Mesmo trade-off de RWX do item anterior.

### 7. SemVer com Conventional Commits + default patch

**Escolha**: Parser de commit messages seguindo Conventional Commits:
- `feat:` / `feat(...):` → minor
- `fix:` / `chore:` / outros prefixos → patch
- `feat!:` / `BREAKING CHANGE:` no footer → major
- Sem prefixo reconhecido → patch (default)

Cada projeto mantém sua versão independente no TiDB. Versão inicial: `0.1.0`.

**Alternativas consideradas**:
- Versão global para todo o monorepo: perde a semântica por projeto.
- Tag no Git: requer push access e adiciona complexidade.

**Rationale**: Conventional Commits é padrão de mercado. Default patch evita que commits sem prefixo bloqueiem o pipeline.

### 8. Retry application-level por projeto (não via nack NATS)

**Escolha**: Builds falhos são retentados internamente pelo mesmo worker, por projeto, até 3 tentativas com backoff exponencial. A mensagem NATS não é nacked para falhas de build. Após 3 falhas, o projeto é marcado como `failure` em `build_records` e o worker continua para os demais projetos do job.

**Rationale**: Re-enfileirar no NATS forçaria re-clone e re-processamento de todos os projetos do job — incluindo os já concluídos com sucesso. Retry application-level é mais eficiente: reutiliza o clone local e reprocessa apenas o projeto que falhou. Ver Decision #13 para detalhes de configuração (AckWait, MaxDelivers, heartbeat).

### 9. Métricas via Datadog DogStatsD

**Escolha**: Emitir métricas via DogStatsD (datadog-go) para o Datadog Agent rodando como DaemonSet.

Métricas principais:
- `build.duration` (histogram, tags: project, language)
- `build.status` (count, tags: project, status=success|failure)
- `build.queue_wait_time` (histogram)
- `build.projects_affected` (gauge)
- `build.retry_count` (count, tags: project)

**Rationale**: Datadog já é a solução de observabilidade do cluster. DogStatsD é leve e non-blocking. Logs estruturados via zap (JSON para stdout) são coletados automaticamente pelo Datadog Agent.

### 10. Build context e Dockerfile via filesystem local do worker

**Escolha**: Com Buildah no worker pod, o build context é o clone local já disponível no filesystem do worker (`/tmp/repo-<job-id>`). O Dockerfile gerado é escrito em um arquivo temporário no disco local imediatamente antes de chamar `buildah bud`, e deletado após a conclusão. Não são necessários init containers, ConfigMaps, clones secundários ou volumes adicionais.

**Rationale**: A decisão anterior (init container + ConfigMap) foi necessária para entregar build context e Dockerfile a pods Kaniko efêmeros isolados do worker. Com Buildah executando no worker, esse problema não existe — o worker já possui o clone e pode escrever o Dockerfile diretamente no filesystem local. A simplificação elimina a gestão de ciclo de vida de ConfigMaps, o segundo clone do repositório, e a dependência de `client-go`.

**Fluxo por projeto**:
1. Dockerfile gerado em memória pelo template engine
2. Dockerfile escrito em `/tmp/dockerfile-<job-id>-<project>`
3. `buildah bud -f /tmp/dockerfile-<job-id>-<project> -t <registry>/<project>:<version> /tmp/repo-<job-id>`
4. `buildah push <registry>/<project>:<version> --authfile <registry-secret-mount>`
5. Arquivo temporário do Dockerfile deletado

**Consequência no build context**: O context continua sendo a raiz do monorepo (`/tmp/repo-<job-id>`), preservando o suporte a shared libs de `libs/` para projetos Go.

### 11. Worker gera o installation token (não o webhook server)

**Escolha**: O webhook server extrai o `installation_id` do payload do webhook e o publica no NATS. O worker gera um installation token fresco a partir das credenciais do GitHub App imediatamente antes de executar o clone.

**Alternativas consideradas**:
- Token gerado no webhook server e embarcado no payload NATS (design original): token expira em 1h; jobs retidos na fila por mais de 1h falham com 401 no clone.
- Worker re-gera token somente em caso de 401 no clone: frágil — detectar 401 vs outros erros de rede requer parsing da saída do git; não resolve token expirado no init container do Kaniko.

**Rationale**: Gerar o token imediatamente antes de usá-lo garante freshness independente do tempo de espera na fila. O worker, que já precisa das credenciais do GitHub App para gerar tokens, é o lugar correto. O webhook server fica mais simples — precisa apenas do webhook secret para validação de assinatura.

**Consequência na separação de credenciais**:
- webhook-server: precisa apenas do webhook secret
- worker: precisa do GitHub App private key + App ID (para gerar tokens)
- O payload NATS passa a conter `installation_id` (inteiro, não expira) em vez de `installation_token`

### 12. Idempotência via two-phase claim com UNIQUE constraint

**Escolha**: A tabela `build_records` tem UNIQUE constraint em `(project, commit_sha)`. O worker tenta inserir um registro `pending` antes de iniciar o build. A inserção é atômica: se retornar `affected rows = 1`, o worker é o dono; se retornar `affected rows = 0` (duplicate), verifica o status existente. Registros `pending` com `claimed_at` mais antigo que um threshold configurável (padrão: 30 minutos) são considerados stale e permitem re-claim via UPDATE condicional.

**Alternativas consideradas**:
- SELECT + INSERT sem constraint (design original): sujeito a TOCTOU — dois workers podem passar pelo SELECT simultaneamente e ambos iniciarem o build.
- SELECT FOR UPDATE pessimista: requer lock de longa duração (builds podem levar 20+ min), causando timeouts e deadlocks no TiDB.
- Consumer exclusivo no NATS: elimina paralelismo para jobs com múltiplos projetos; não resolve re-delivery após crash do worker.

**Rationale**: INSERT com duplicate key check é atômico no TiDB (single-statement transaction). O worker que consegue inserir "owns" o build; concorrentes recebem erro de constraint e descartam. O stale timeout cobre crash de worker + NATS redelivery sem deixar projetos bloqueados permanentemente.

**Schema `build_records`**:
```sql
CREATE TABLE build_records (
  id         BIGINT PRIMARY KEY AUTO_INCREMENT,
  project    VARCHAR(255) NOT NULL,
  commit_sha CHAR(40)     NOT NULL,
  status     ENUM('pending', 'success', 'failure') NOT NULL DEFAULT 'pending',
  claimed_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_project_sha (project, commit_sha)
);
```

**Fluxo de claim**:
1. `INSERT ... ON DUPLICATE KEY UPDATE id = id`
2. `affected = 1` → prossegue com o build
3. `affected = 0` → lê status: `success`/`failure` → skip; `pending` recente → skip; `pending` stale → tenta UPDATE condicional de `claimed_at` (se `affected = 1`, re-claimed com sucesso; se `affected = 0`, outro worker ganhou a corrida do re-claim → skip)

### 13. Retry application-level por projeto com heartbeat NATS

**Escolha**: Retries de build são tratados em nível de aplicação: cada projeto é retentado internamente pelo mesmo worker até 3 vezes com backoff exponencial antes de ser marcado como falha permanente. A mensagem NATS é mantida em processamento via chamadas periódicas a `msg.InProgress()` (a cada 2 minutos) durante toda a duração do job. O consumer JetStream é configurado com `MaxDelivers: 3` exclusivamente para cobrir crashes de worker — não para retries de build.

**Alternativas consideradas**:
- Retry via nack NATS (reentrega pelo JetStream): reentrega o job inteiro para qualquer worker disponível, forçando re-clone e re-processamento de todos os projetos do job — inclusive os já concluídos com sucesso. Com idempotência, os projetos bem-sucedidos seriam pulados, mas o custo de re-clone e re-execução do `nx affected` é alto e desnecessário.
- Retry via nack sem heartbeat: inviável — `AckWait` precisaria cobrir o pior caso total do job (potencialmente horas), impedindo reentrega rápida em crashes reais.

**Rationale**: Retry application-level roda no mesmo worker, reutiliza o clone local e reprocessa apenas o projeto que falhou. O heartbeat `msg.InProgress()` mantém o `AckWait` curto (5 minutos) sem causar reentregas falsas durante processamento legítimo longo. A separação entre "retry de build" (aplicação, max 3 por projeto) e "recovery de crash de worker" (NATS `MaxDelivers: 3` por job) torna a semântica inequívoca.

**Configuração do consumer NATS**:
- `AckWait`: 5 minutos
- `MaxDelivers`: 3 (para recovery de crash — independente do número de retries de build)
- Heartbeat: `msg.InProgress()` a cada 2 minutos enquanto o job estiver em processamento

### 14. SHA avança sempre; builds com falha permanente requerem nova mudança de código

**Escolha**: O `last_processed_sha` é sempre atualizado para o SHA do push após o processamento completo, independentemente do resultado dos builds individuais. Projetos com falha permanente ficam marcados como `failure` em `build_records` e não têm imagem publicada para aquele SHA. A reconstrução automática ocorre naturalmente quando o próximo push incluir aquele projeto no diff do `nx affected`.

**Alternativas consideradas**:
- SHA só avança se todos os builds tiverem sucesso: auto-curativo para falhas transitórias, mas projetos com erro persistente de código bloqueiam o avanço do SHA para todo o repositório — cada push futuro acumula mais projetos no `nx affected`, crescendo o diff sem limite.
- SHA por projeto (`last_sha` individual): tracking independente por projeto; complexidade alta e necessidade de resolver o SHA base mínimo entre todos os projetos para o `nx affected`.

**Rationale**: SHA por push é a semântica correta — rastreia "qual push foi processado", não "quais projetos tiveram sucesso". A reconstrução ao próximo commit que toca o projeto é o fluxo padrão de CI/CD. Alertas Datadog em `build.status = failure` notificam operadores para agir (corrigir o código). Não há builds perdidos: projetos com falha permanente aparecem novamente no `nx affected` assim que houver uma mudança no código.

**Implicação para SemVer**: Se um projeto falha permanentemente em um push, sua versão não é incrementada. Quando corrigido em um push futuro, o bump reflete apenas os commits daquele push — podendo subestimar a mudança real acumulada. Esse é um trade-off aceito; a alternativa de acumular bump pendente adiciona estado e complexidade incompatíveis com o restante do design.

## Risks / Trade-offs

- **PVC ReadWriteMany para nx-cache** → Verificar disponibilidade de storage class RWX no cluster. O buildah-storage PVC pode ser RWO por worker (cada pod tem sua própria store de layers), eliminando o requisito de RWX para builds.
- **Worker precisa de Node.js + Nx instalados para rodar `nx affected`** → Imagem do worker será pesada. Mitigação: multi-stage build com camadas separadas para Go runtime, Node.js/Nx e buildah.
- **Buildah no worker requer capabilities de kernel** → `buildah bud` com overlay filesystem precisa de `CAP_SETUID` e `CAP_SETFCAP`. Mitigação: configurar SecurityContext do pod; alternativa: `--storage-driver vfs` para operação totalmente rootless sem capabilities (mais lento, sem cache de layers por diff).
- **Builds paralelos no mesmo pod compartilham filesystem e CPU** → Múltiplos processos `buildah bud` simultâneos no mesmo worker disputam I/O de disco e CPU. Mitigação: semáforo de concorrência configurável via Viper; dimensionar recursos do pod de acordo.
- **NATS JetStream message replay em caso de crash** → At-least-once delivery pode causar builds duplicados. Mitigação: idempotência baseada em SHA+project no TiDB (verificar se versão já foi buildada).
- **Conventional Commits dependem de disciplina dos devs** → Default patch garante que builds nunca fiquem bloqueados, mesmo com mensagens fora do padrão.
- **TiDB como ponto de falha** → Se TiDB estiver indisponível, workers não conseguem resolver versões. Mitigação: health checks + retry com backoff na conexão.
