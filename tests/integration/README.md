# Integration Tests - Robot Framework

Este diretório contém testes de integração end-to-end para o OCI Build System usando Robot Framework.

## Estrutura

```
tests/integration/
├── requirements.txt          # Dependências Python/Robot Framework
├── README.md                 # Este arquivo
├── resources/
│   ├── keywords.robot        # Keywords customizadas reutilizáveis
│   └── variables.robot       # Variáveis de teste
├── fixtures/
│   ├── sample-java-repo/     # Repositório de teste Java
│   ├── sample-dotnet-repo/   # Repositório de teste .NET
│   └── sample-go-repo/       # Repositório de teste Go
├── webhook.robot             # Testes de webhook
├── build.robot               # Testes de build
└── api.robot                 # Testes de API
```

## Pré-requisitos

1. Python 3.8+
2. Docker e Docker Compose
3. Sistema OCI Build rodando localmente

## Instalação

```bash
cd tests/integration
pip install -r requirements.txt
```

## Executando os Testes

### Subir o ambiente

```bash
docker-compose up -d
```

### Executar todos os testes

```bash
robot tests/integration/
```

### Executar suite específica

```bash
robot tests/integration/webhook.robot
robot tests/integration/build.robot
robot tests/integration/api.robot
```

### Executar teste específico

```bash
robot -t "Send Valid Webhook" tests/integration/webhook.robot
```

### Gerar relatório detalhado

```bash
robot --outputdir results tests/integration/
```

## Variáveis de Ambiente

As seguintes variáveis podem ser configuradas:

- `API_BASE_URL`: URL base da API (padrão: http://localhost:8080)
- `GITHUB_SECRET`: Secret para validação de webhooks (padrão: test-secret-key)
- `AUTH_TOKEN`: Token de autenticação para endpoints (padrão: test-auth-token)

Exemplo:

```bash
robot --variable API_BASE_URL:http://api:8080 tests/integration/
```

## Fixtures de Teste

Os repositórios de teste em `fixtures/` são repositórios mínimos que simulam projetos reais:

- **sample-java-repo**: Projeto Maven com Dockerfile
- **sample-dotnet-repo**: Projeto .NET com Dockerfile
- **sample-go-repo**: Projeto Go com Dockerfile

Estes repositórios são usados para testar builds completos end-to-end.

## Troubleshooting

### Testes falhando com timeout

Aumente o timeout nos testes ou verifique se os serviços estão rodando:

```bash
docker-compose ps
docker-compose logs api-service
docker-compose logs worker-service
```

### Erro de conexão com API

Verifique se a API está acessível:

```bash
curl http://localhost:8080/health
```

### Erro de autenticação

Verifique se o token de autenticação está configurado corretamente no docker-compose.yml
