# Guia de Deployment

Este documento descreve como fazer deploy do OCI Build System em ambientes de produção.

## 📋 Requisitos de Infraestrutura

### Hardware Mínimo

**API Service**:
- CPU: 2 cores
- RAM: 2 GB
- Disco: 10 GB

**Worker Service** (por instância):
- CPU: 4 cores
- RAM: 8 GB
- Disco: 100 GB (para cache de código e dependências)

**NATS Server**:
- CPU: 2 cores
- RAM: 4 GB
- Disco: 20 GB (para persistência de mensagens)

### Hardware Recomendado (Produção)

**API Service**:
- CPU: 4 cores
- RAM: 4 GB
- Disco: 20 GB
- Réplicas: 2+ (para alta disponibilidade)

**Worker Service**:
- CPU: 8 cores
- RAM: 16 GB
- Disco: 500 GB SSD
- Réplicas: 3+ (baseado na carga)

**NATS Cluster**:
- CPU: 4 cores por nó
- RAM: 8 GB por nó
- Disco: 100 GB SSD por nó
- Nós: 3 (para quorum)

### Software

- Docker 24.0+
- Kubernetes 1.27+ (para deployment em cluster)
- Buildah 1.30+ (instalado nos workers)
- Sistema operacional: Linux (Ubuntu 22.04 LTS recomendado)

### Rede

- Portas necessárias:
  - `8080`: API Service (HTTP)
  - `4222`: NATS Client Port
  - `6222`: NATS Cluster Port
  - `8222`: NATS Monitoring Port

## 🐳 Deployment com Docker Compose

### 1. Preparar Ambiente

```bash
# Criar diretórios para volumes
sudo mkdir -p /var/oci-build/{cache,logs,data}
sudo chown -R 1000:1000 /var/oci-build

# Criar arquivo de configuração
cat > .env << EOF
GITHUB_WEBHOOK_SECRET=seu-secret-aqui
NATS_URL=nats://nats:4222
LOG_LEVEL=info
WORKER_POOL_SIZE=5
EOF
```

### 2. Configurar Docker Compose

**docker-compose.prod.yml**:
```yaml
version: '3.8'

services:
  nats:
    image: nats:2.10-alpine
    container_name: oci-build-nats
    command: >
      -js
      -m 8222
      -sd /data
      --cluster_name oci-build-cluster
    ports:
      - "4222:4222"
      - "8222:8222"
    volumes:
      - /var/oci-build/data:/data
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8222/healthz"]
      interval: 10s
      timeout: 5s
      retries: 3

  api-service:
    image: oci-build/api-service:latest
    container_name: oci-build-api
    ports:
      - "8080:8080"
    environment:
      - NATS_URL=${NATS_URL}
      - GITHUB_WEBHOOK_SECRET=${GITHUB_WEBHOOK_SECRET}
      - LOG_LEVEL=${LOG_LEVEL}
    volumes:
      - /var/oci-build/logs:/var/log/oci-build
    depends_on:
      nats:
        condition: service_healthy
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3

  worker-service:
    image: oci-build/worker-service:latest
    environment:
      - NATS_URL=${NATS_URL}
      - LOG_LEVEL=${LOG_LEVEL}
      - WORKER_POOL_SIZE=${WORKER_POOL_SIZE}
    volumes:
      - /var/oci-build/cache:/var/cache/oci-build
      - /var/oci-build/logs:/var/log/oci-build
      - /var/run/docker.sock:/var/run/docker.sock
    depends_on:
      nats:
        condition: service_healthy
    restart: unless-stopped
    deploy:
      replicas: 3
      resources:
        limits:
          cpus: '4'
          memory: 8G
        reservations:
          cpus: '2'
          memory: 4G
```

### 3. Deploy

```bash
# Build das imagens
docker-compose -f docker-compose.prod.yml build

# Iniciar serviços
docker-compose -f docker-compose.prod.yml up -d

# Verificar status
docker-compose -f docker-compose.prod.yml ps

# Verificar logs
docker-compose -f docker-compose.prod.yml logs -f
```

### 4. Configurar Reverse Proxy (Nginx)

```nginx
upstream api_backend {
    least_conn;
    server localhost:8080 max_fails=3 fail_timeout=30s;
}

server {
    listen 80;
    server_name build.example.com;

    # Redirecionar para HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name build.example.com;

    ssl_certificate /etc/ssl/certs/build.example.com.crt;
    ssl_certificate_key /etc/ssl/private/build.example.com.key;

    # Configurações SSL
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # Logs
    access_log /var/log/nginx/oci-build-access.log;
    error_log /var/log/nginx/oci-build-error.log;

    # Webhook endpoint
    location /webhook {
        proxy_pass http://api_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Timeout para webhooks
        proxy_read_timeout 30s;
        proxy_connect_timeout 10s;
    }

    # API endpoints
    location /builds {
        proxy_pass http://api_backend;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # Autenticação
        proxy_set_header Authorization $http_authorization;
    }

    # Health check
    location /health {
        proxy_pass http://api_backend;
        access_log off;
    }
}
```

## ☸️ Deployment com Kubernetes

### 1. Criar Namespace

```bash
kubectl create namespace oci-build
```

### 2. Configurar Secrets

```bash
# Criar secret para webhook
kubectl create secret generic github-webhook \
  --from-literal=secret=seu-secret-aqui \
  -n oci-build

# Criar secret para autenticação da API
kubectl create secret generic api-auth \
  --from-literal=token=seu-token-aqui \
  -n oci-build
```

### 3. Deploy NATS

**nats-deployment.yaml**:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: nats
  namespace: oci-build
spec:
  selector:
    app: nats
  ports:
    - name: client
      port: 4222
      targetPort: 4222
    - name: monitoring
      port: 8222
      targetPort: 8222
  clusterIP: None
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: nats
  namespace: oci-build
spec:
  serviceName: nats
  replicas: 3
  selector:
    matchLabels:
      app: nats
  template:
    metadata:
      labels:
        app: nats
    spec:
      containers:
      - name: nats
        image: nats:2.10-alpine
        args:
          - "-js"
          - "-m"
          - "8222"
          - "-sd"
          - "/data"
          - "--cluster_name"
          - "oci-build-cluster"
        ports:
        - containerPort: 4222
          name: client
        - containerPort: 8222
          name: monitoring
        volumeMounts:
        - name: data
          mountPath: /data
        resources:
          requests:
            cpu: 2
            memory: 4Gi
          limits:
            cpu: 4
            memory: 8Gi
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 100Gi
```

### 4. Deploy API Service

**api-service-deployment.yaml**:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: api-service
  namespace: oci-build
spec:
  selector:
    app: api-service
  ports:
    - port: 80
      targetPort: 8080
  type: LoadBalancer
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-service
  namespace: oci-build
spec:
  replicas: 3
  selector:
    matchLabels:
      app: api-service
  template:
    metadata:
      labels:
        app: api-service
    spec:
      containers:
      - name: api-service
        image: oci-build/api-service:latest
        ports:
        - containerPort: 8080
        env:
        - name: NATS_URL
          value: "nats://nats:4222"
        - name: GITHUB_WEBHOOK_SECRET
          valueFrom:
            secretKeyRef:
              name: github-webhook
              key: secret
        - name: LOG_LEVEL
          value: "info"
        resources:
          requests:
            cpu: 1
            memory: 2Gi
          limits:
            cpu: 2
            memory: 4Gi
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 5
```

### 5. Deploy Worker Service

**worker-service-deployment.yaml**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: worker-service
  namespace: oci-build
spec:
  replicas: 5
  selector:
    matchLabels:
      app: worker-service
  template:
    metadata:
      labels:
        app: worker-service
    spec:
      containers:
      - name: worker-service
        image: oci-build/worker-service:latest
        env:
        - name: NATS_URL
          value: "nats://nats:4222"
        - name: LOG_LEVEL
          value: "info"
        - name: WORKER_POOL_SIZE
          value: "5"
        volumeMounts:
        - name: cache
          mountPath: /var/cache/oci-build
        - name: docker-sock
          mountPath: /var/run/docker.sock
        resources:
          requests:
            cpu: 2
            memory: 4Gi
          limits:
            cpu: 8
            memory: 16Gi
      volumes:
      - name: cache
        persistentVolumeClaim:
          claimName: worker-cache
      - name: docker-sock
        hostPath:
          path: /var/run/docker.sock
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: worker-cache
  namespace: oci-build
spec:
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 500Gi
```

### 6. Deploy

```bash
# Aplicar configurações
kubectl apply -f nats-deployment.yaml
kubectl apply -f api-service-deployment.yaml
kubectl apply -f worker-service-deployment.yaml

# Verificar status
kubectl get pods -n oci-build
kubectl get services -n oci-build

# Ver logs
kubectl logs -f deployment/api-service -n oci-build
kubectl logs -f deployment/worker-service -n oci-build
```

### 7. Configurar Ingress

**ingress.yaml**:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: oci-build-ingress
  namespace: oci-build
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - build.example.com
    secretName: oci-build-tls
  rules:
  - host: build.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: api-service
            port:
              number: 80
```

## 🔧 Configuração de NATS em Produção

### Cluster NATS

Para alta disponibilidade, configure um cluster NATS com 3 nós:

**nats-cluster.conf**:
```
port: 4222
http_port: 8222

jetstream {
  store_dir: /data
  max_memory_store: 4GB
  max_file_store: 100GB
}

cluster {
  name: oci-build-cluster
  listen: 0.0.0.0:6222
  
  routes = [
    nats://nats-0.nats:6222
    nats://nats-1.nats:6222
    nats://nats-2.nats:6222
  ]
}

# Autenticação
authorization {
  users = [
    {
      user: "api-service"
      password: "$2a$11$..."
      permissions: {
        publish: ["builds.>"]
        subscribe: ["builds.>"]
      }
    }
    {
      user: "worker-service"
      password: "$2a$11$..."
      permissions: {
        publish: ["builds.>"]
        subscribe: ["builds.>"]
      }
    }
  ]
}

# Limites
max_connections: 1000
max_payload: 10MB
```

### Persistência de Mensagens

Configure JetStream para persistência:

```bash
# Criar stream para builds
nats stream add BUILDS \
  --subjects "builds.*" \
  --storage file \
  --retention limits \
  --max-msgs 10000 \
  --max-age 7d
```

## 📊 Monitoramento e Observabilidade

### Prometheus

**prometheus.yml**:
```yaml
scrape_configs:
  - job_name: 'nats'
    static_configs:
      - targets: ['nats:8222']
    metrics_path: /metrics

  - job_name: 'api-service'
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
            - oci-build
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_label_app]
        regex: api-service
        action: keep
```

### Grafana Dashboards

Importe dashboards para monitorar:
- Taxa de builds (sucesso/falha)
- Duração média de builds
- Uso de cache
- Fila de builds
- Métricas de NATS

### Alertas

Configure alertas para:
- Taxa de falha > 10%
- Fila de builds > 100
- Uso de disco > 80%
- Serviços indisponíveis

**alertmanager.yml**:
```yaml
route:
  receiver: 'team-notifications'
  group_by: ['alertname', 'cluster']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h

receivers:
  - name: 'team-notifications'
    slack_configs:
      - api_url: 'https://hooks.slack.com/services/...'
        channel: '#oci-build-alerts'
        title: 'OCI Build Alert'
        text: '{{ range .Alerts }}{{ .Annotations.description }}{{ end }}'
```

## 🔐 Segurança

### TLS/SSL

Configure certificados SSL para:
- API Service (HTTPS)
- NATS (TLS)

```bash
# Gerar certificados com Let's Encrypt
certbot certonly --standalone -d build.example.com
```

### Autenticação

Configure autenticação para:
- Webhooks GitHub (HMAC-SHA256)
- API REST (Bearer tokens)
- NATS (usuário/senha)

### Network Policies

Restrinja comunicação entre pods:

**network-policy.yaml**:
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: worker-policy
  namespace: oci-build
spec:
  podSelector:
    matchLabels:
      app: worker-service
  policyTypes:
  - Ingress
  - Egress
  ingress: []
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: nats
    ports:
    - protocol: TCP
      port: 4222
```

## 🔄 Backup e Recuperação

### Backup de Dados

```bash
# Backup do cache
tar -czf cache-backup-$(date +%Y%m%d).tar.gz /var/oci-build/cache

# Backup de logs
tar -czf logs-backup-$(date +%Y%m%d).tar.gz /var/oci-build/logs

# Backup de dados NATS
kubectl exec -n oci-build nats-0 -- tar -czf /tmp/nats-backup.tar.gz /data
kubectl cp oci-build/nats-0:/tmp/nats-backup.tar.gz ./nats-backup-$(date +%Y%m%d).tar.gz
```

### Recuperação

```bash
# Restaurar cache
tar -xzf cache-backup-20240101.tar.gz -C /var/oci-build/

# Restaurar NATS
kubectl cp nats-backup-20240101.tar.gz oci-build/nats-0:/tmp/
kubectl exec -n oci-build nats-0 -- tar -xzf /tmp/nats-backup.tar.gz -C /
kubectl rollout restart statefulset/nats -n oci-build
```

## 📈 Escalabilidade

### Escalar Workers

```bash
# Docker Compose
docker-compose -f docker-compose.prod.yml up -d --scale worker-service=10

# Kubernetes
kubectl scale deployment worker-service --replicas=10 -n oci-build
```

### Auto-scaling (Kubernetes)

**hpa.yaml**:
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: worker-service-hpa
  namespace: oci-build
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: worker-service
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

## 🔍 Troubleshooting em Produção

### Logs Centralizados

Configure agregação de logs com ELK Stack ou Loki:

```bash
# Ver logs de todos os workers
kubectl logs -l app=worker-service -n oci-build --tail=100

# Buscar erros
kubectl logs -l app=worker-service -n oci-build | grep ERROR
```

### Métricas de Performance

```bash
# Ver uso de recursos
kubectl top pods -n oci-build

# Ver eventos
kubectl get events -n oci-build --sort-by='.lastTimestamp'
```

## 📞 Suporte

Para problemas em produção:
- Verifique logs e métricas
- Consulte alertas do Prometheus
- Abra issue no GitHub com logs relevantes
- Entre em contato com a equipe de DevOps
