# OCI Build System - PowerShell Build Script
# Equivalente ao Makefile para Windows

param(
    [Parameter(Position=0)]
    [string]$Command = "help"
)

$ErrorActionPreference = "Stop"

# Cores
function Write-Success { Write-Host $args -ForegroundColor Green }
function Write-Info { Write-Host $args -ForegroundColor Cyan }
function Write-Warning { Write-Host $args -ForegroundColor Yellow }
function Write-Error { Write-Host $args -ForegroundColor Red }

function Show-Help {
    Write-Success "`nOCI Build System - Comandos disponíveis:`n"
    Write-Host "  help              " -ForegroundColor Yellow -NoNewline; Write-Host "Mostra esta mensagem de ajuda"
    Write-Host "  install           " -ForegroundColor Yellow -NoNewline; Write-Host "Instala todas as dependências (npm e go)"
    Write-Host "  build             " -ForegroundColor Yellow -NoNewline; Write-Host "Executa build de todos os projetos usando NX"
    Write-Host "  build-affected    " -ForegroundColor Yellow -NoNewline; Write-Host "Executa build apenas dos projetos afetados"
    Write-Host "  build-shared      " -ForegroundColor Yellow -NoNewline; Write-Host "Executa build da biblioteca shared"
    Write-Host "  test              " -ForegroundColor Yellow -NoNewline; Write-Host "Executa testes de todos os projetos"
    Write-Host "  test-affected     " -ForegroundColor Yellow -NoNewline; Write-Host "Executa testes dos projetos afetados"
    Write-Host "  test-shared       " -ForegroundColor Yellow -NoNewline; Write-Host "Executa testes da biblioteca shared"
    Write-Host "  test-coverage     " -ForegroundColor Yellow -NoNewline; Write-Host "Executa testes com cobertura"
    Write-Host "  run               " -ForegroundColor Yellow -NoNewline; Write-Host "Inicia todos os serviços usando docker-compose"
    Write-Host "  run-nats          " -ForegroundColor Yellow -NoNewline; Write-Host "Inicia apenas o NATS"
    Write-Host "  stop              " -ForegroundColor Yellow -NoNewline; Write-Host "Para todos os serviços"
    Write-Host "  clean             " -ForegroundColor Yellow -NoNewline; Write-Host "Remove arquivos de build e cache"
    Write-Host "  format            " -ForegroundColor Yellow -NoNewline; Write-Host "Formata o código Go"
    Write-Host "  graph             " -ForegroundColor Yellow -NoNewline; Write-Host "Mostra o gráfico de dependências"
    Write-Host "  ci                " -ForegroundColor Yellow -NoNewline; Write-Host "Executa pipeline completo de CI"
    Write-Host ""
}

function Install-Dependencies {
    Write-Success "Instalando dependências npm..."
    npm install
    Write-Success "Instalando dependências Go..."
    Push-Location libs/shared
    go mod download
    Pop-Location
    Write-Success "Dependências instaladas com sucesso!"
}

function Build-All {
    Write-Success "Executando build de todos os projetos..."
    npm exec nx run-many -- --target=build --all --parallel=3
    Write-Success "Build concluído com sucesso!"
}

function Build-Affected {
    Write-Success "Executando build dos projetos afetados..."
    npm exec nx affected -- --target=build --parallel=3
    Write-Success "Build dos projetos afetados concluído!"
}

function Build-Shared {
    Write-Success "Executando build da biblioteca shared..."
    npm exec nx run shared:build
    Write-Success "Build da shared concluído!"
}

function Test-All {
    Write-Success "Executando testes de todos os projetos..."
    npm exec nx run-many -- --target=test --all --parallel=3
    Write-Success "Testes concluídos!"
}

function Test-Affected {
    Write-Success "Executando testes dos projetos afetados..."
    npm exec nx affected -- --target=test --parallel=3
    Write-Success "Testes dos projetos afetados concluídos!"
}

function Test-Shared {
    Write-Success "Executando testes da biblioteca shared..."
    npm exec nx run shared:test
    Write-Success "Testes da shared concluídos!"
}

function Test-Coverage {
    Write-Success "Executando testes com cobertura..."
    npm exec nx run-many -- --target=test --all --parallel=3 --coverage
    Write-Success "Testes com cobertura concluídos!"
}

function Run-Services {
    Write-Success "Iniciando todos os serviços..."
    docker-compose up -d
    Write-Success "Serviços iniciados! Use 'docker-compose logs -f' para ver os logs."
}

function Run-Nats {
    Write-Success "Iniciando NATS..."
    docker-compose up -d nats
    Write-Success "NATS iniciado!"
}

function Stop-Services {
    Write-Warning "Parando todos os serviços..."
    docker-compose down
    Write-Success "Serviços parados!"
}

function Clean-Build {
    Write-Warning "Limpando arquivos de build e cache..."
    npm exec nx reset
    if (Test-Path dist) { Remove-Item -Recurse -Force dist }
    if (Test-Path .nx/cache) { Remove-Item -Recurse -Force .nx/cache }
    Get-ChildItem -Recurse -Filter "*.test" | Remove-Item -Force
    Write-Success "Limpeza concluída!"
}

function Format-Code {
    Write-Success "Formatando código Go..."
    Get-ChildItem -Recurse -Filter "*.go" | Where-Object { $_.FullName -notlike "*node_modules*" } | ForEach-Object {
        gofmt -w $_.FullName
    }
    Write-Success "Código formatado!"
}

function Show-Graph {
    Write-Success "Abrindo gráfico de dependências..."
    npm exec nx graph
}

function Run-CI {
    Write-Success "Executando pipeline de CI..."
    Install-Dependencies
    Build-All
    Test-All
    Write-Success "Pipeline de CI concluído com sucesso!"
}

# Executar comando
switch ($Command.ToLower()) {
    "help" { Show-Help }
    "install" { Install-Dependencies }
    "build" { Build-All }
    "build-affected" { Build-Affected }
    "build-shared" { Build-Shared }
    "test" { Test-All }
    "test-affected" { Test-Affected }
    "test-shared" { Test-Shared }
    "test-coverage" { Test-Coverage }
    "run" { Run-Services }
    "run-nats" { Run-Nats }
    "stop" { Stop-Services }
    "clean" { Clean-Build }
    "format" { Format-Code }
    "graph" { Show-Graph }
    "ci" { Run-CI }
    default {
        Write-Error "Comando desconhecido: $Command"
        Show-Help
        exit 1
    }
}
