#!/bin/bash
set -e

# Cores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuração
MAX_RETRIES=${MAX_RETRIES:-30}
RETRY_INTERVAL=${RETRY_INTERVAL:-2}
API_URL=${API_URL:-http://localhost:8080}
NATS_URL=${NATS_URL:-http://localhost:8222}

# Função para calcular backoff exponencial
calculate_backoff() {
    local attempt=$1
    local base_interval=$RETRY_INTERVAL
    
    # Backoff exponencial: base * 2^(attempt/5)
    # Limitado a 10 segundos máximo
    local backoff=$(awk "BEGIN {print int($base_interval * (1.2 ^ ($attempt / 3)))}")
    if [ $backoff -gt 10 ]; then
        backoff=10
    fi
    echo $backoff
}

# Função para verificar se um serviço está pronto
check_service() {
    local service_name=$1
    local health_url=$2
    local retries=0
    
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${YELLOW}⏳ Checking ${service_name}...${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    
    while [ $retries -lt $MAX_RETRIES ]; do
        # Tentar conectar ao serviço
        if curl -f -s -o /dev/null --max-time 5 "$health_url" 2>/dev/null; then
            echo -e "${GREEN}✓ ${service_name} is ready!${NC}"
            echo -e "${GREEN}  └─ Health check passed at ${health_url}${NC}"
            echo ""
            return 0
        fi
        
        retries=$((retries + 1))
        
        # Calcular tempo de espera com backoff
        local wait_time=$(calculate_backoff $retries)
        
        # Mostrar progresso
        local progress=$((retries * 100 / MAX_RETRIES))
        echo -e "${YELLOW}  ⌛ Attempt ${retries}/${MAX_RETRIES} (${progress}%) - ${service_name} not ready yet${NC}"
        echo -e "${YELLOW}     Waiting ${wait_time}s before retry...${NC}"
        
        sleep $wait_time
    done
    
    echo -e "${RED}✗ ${service_name} failed to become ready after ${MAX_RETRIES} attempts${NC}"
    echo -e "${RED}  └─ Health check URL: ${health_url}${NC}"
    echo -e "${RED}  └─ Total time waited: ~$((MAX_RETRIES * RETRY_INTERVAL))s${NC}"
    echo ""
    return 1
}

# Banner inicial
echo ""
echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  ${YELLOW}Waiting for Services to be Ready${BLUE}  ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}Configuration:${NC}"
echo -e "  • Max retries: ${MAX_RETRIES}"
echo -e "  • Base retry interval: ${RETRY_INTERVAL}s"
echo -e "  • API URL: ${API_URL}"
echo -e "  • NATS URL: ${NATS_URL}"
echo ""

# Timestamp de início
start_time=$(date +%s)

# Verificar NATS
if ! check_service "NATS" "${NATS_URL}/healthz"; then
    echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${RED}❌ NATS is not ready. Exiting.${NC}"
    echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo -e "${YELLOW}Troubleshooting tips:${NC}"
    echo -e "  1. Check if NATS container is running: ${BLUE}docker-compose ps nats${NC}"
    echo -e "  2. Check NATS logs: ${BLUE}docker-compose logs nats${NC}"
    echo -e "  3. Verify NATS URL is correct: ${NATS_URL}"
    echo ""
    exit 1
fi

# Verificar API Service
if ! check_service "API Service" "${API_URL}/health"; then
    echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${RED}❌ API Service is not ready. Exiting.${NC}"
    echo -e "${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo ""
    echo -e "${YELLOW}Troubleshooting tips:${NC}"
    echo -e "  1. Check if API Service container is running: ${BLUE}docker-compose ps api-service${NC}"
    echo -e "  2. Check API Service logs: ${BLUE}docker-compose logs api-service${NC}"
    echo -e "  3. Verify API URL is correct: ${API_URL}"
    echo -e "  4. Check if required environment variables are set (GITHUB_WEBHOOK_SECRET, API_AUTH_TOKEN)"
    echo ""
    exit 1
fi

# Timestamp de fim
end_time=$(date +%s)
elapsed=$((end_time - start_time))

# Banner de sucesso
echo -e "${GREEN}╔════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║     ${YELLOW}✓ All Services are Ready!${GREEN}       ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════╝${NC}"
echo ""
echo -e "${GREEN}Summary:${NC}"
echo -e "  • NATS: ${GREEN}✓ Ready${NC}"
echo -e "  • API Service: ${GREEN}✓ Ready${NC}"
echo -e "  • Total time: ${elapsed}s"
echo ""
echo -e "${BLUE}You can now run your integration tests!${NC}"
echo ""

exit 0
