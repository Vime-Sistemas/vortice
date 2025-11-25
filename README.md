# Vortice — lightweight Go load balancer

## Visão geral

Vortice é um load-balancer escrito em Go, focado em simplicidade, testabilidade e uso em ambientes de desenvolvimento ou cargas leves. Ele fornece recursos que facilitam o desenvolvimento de aplicações distribuídas e testes locais.

Features principais:
- Algoritmos de balanceamento: `round_robin`, `least_conn`, `random`, `ip_hash`.
- Health checks periódicos com configuração inicial imediata.
- Rate limiting por backend (global e por-backend posicional).
- Capacidade de iniciar backends locais embutidos para testes rápidos (`START_LOCAL_BACKENDS`).
- `ip_hash` configurável para usar `RemoteAddr` ou um header (ex: `X-Forwarded-For`).
- Handler de erro personalizável no proxy; respostas 503/429 quando aplicável.
- API programática: `ServerPool` e `NewBackend` podem ser usadas como biblioteca em outros projetos.

O objetivo é ser leve e simples de integrar em pipelines de desenvolvimento e testes, sem prescrição de infra.

## Build

No Windows/PowerShell (no diretório do projeto):

```powershell
go build ./cmd/main.go
# ou
go build -o vortice.exe ./cmd/main.go
```

## Execução

Exemplo simples (usar `.env.local` ou variáveis de ambiente):

```powershell
# usando variáveis temporárias no PowerShell
$env:BACKEND_URLS = 'http://localhost:8081,http://localhost:8082'
$env:APP_PORT = '8080'
./main.exe
```

## Configuração (variáveis de ambiente)

- `BACKEND_URLS` — lista de URLs separadas por vírgula. Ex: `http://host1:8081,http://host2:8082`.
- `APP_PORT` — porta onde o proxy escuta (padrão `8080`).

Opções para iniciar backends locais (útil para desenvolvimento):
- `START_LOCAL_BACKENDS` — `true|false`. Quando `true`, o processo inicia N servidores locais simples.
- `LOCAL_BACKEND_START_PORT` — porta inicial (padrão `8081`).
- `LOCAL_BACKEND_COUNT` — quantos backends iniciar (padrão `3`).
- `LOCAL_BACKEND_FORCE` — `true|false`. Quando `true`, os backends locais substituem `BACKEND_URLS` configurados.

## Algoritmos de balanceamento
- `LOAD_BALANCER_ALGO` — `round_robin` (padrão), `least_conn`, `random`, `ip_hash`.
  - `round_robin`: rotaciona entre backends ativos.
  - `least_conn`: escolhe o backend com menos conexões ativas.
  - `random`: escolhe um backend ativo aleatoriamente.
  - `ip_hash`: escolhe backend baseado em hash do IP do cliente ou de um header configurado.
- `IP_HASH_HEADER` — nome do header a ser usado para `ip_hash` (ex: `X-Forwarded-For`). Se vazio, usa `RemoteAddr`.

## Rate limiting
- `RATE_LIMIT_RPS` — taxa global (requests per second) por backend (0 = desabilitado).
- `RATE_LIMIT_BURST` — burst do limiter (padrão 1).
- `BACKEND_RATE_LIMITS` — opcional, define limites por backend na ordem da lista `BACKEND_URLS`.
  - Formato: `rps/burst,rps/burst,...` ou apenas `rps,rps`.
  - Exemplo: `BACKEND_RATE_LIMITS=10/5,0/0,2/1` — primeiro backend 10rps/5burst, segundo sem limit, terceiro 2rps/1burst.

## Exemplos práticos

1) Iniciar apenas como proxy (just distribute):

```powershell
$env:BACKEND_URLS='http://10.0.0.1:8080,http://10.0.0.2:8080'
$env:APP_PORT='8080'
./main.exe
```

2) Subir backends locais automaticamente (dev):

```powershell
$env:START_LOCAL_BACKENDS='true'
$env:LOCAL_BACKEND_START_PORT='8081'
$env:LOCAL_BACKEND_COUNT='3'
$env:LOCAL_BACKEND_FORCE='true' # substitui BACKEND_URLS
./main.exe
```

3) Usar `least_conn` com rate limit global por backend:

```powershell
$env:BACKEND_URLS='http://localhost:8081,http://localhost:8082'
$env:LOAD_BALANCER_ALGO='least_conn'
$env:RATE_LIMIT_RPS='5'
$env:RATE_LIMIT_BURST='2'
./main.exe
```

4) Limites por backend (posicionais):

```powershell
$env:BACKEND_URLS='http://localhost:8081,http://localhost:8082'
$env:BACKEND_RATE_LIMITS='10/5,0/0' # primeiro com limit, segundo sem limit
./main.exe
```

5) `ip_hash` usando `X-Forwarded-For`:

```powershell
$env:LOAD_BALANCER_ALGO='ip_hash'
$env:IP_HASH_HEADER='X-Forwarded-For'
./main.exe
```

## Testes

```powershell
go test ./domain -v
```

## Usando como módulo

Se você preferir usar o load balancer como uma biblioteca em outro projeto, importe os pacotes usando o módulo público:

```go
import "github.com/Vime-Sistemas/vortice/domain"
```

Exemplo em `go.mod` do consumidor:

```
module your/project

require github.com/Vime-Sistemas/vortice v0.0.0

# se estiver desenvolvendo localmente, use replace:
replace github.com/Vime-Sistemas/vortice => ../path/to/vortice
```

Exemplo de uso programático está em `examples/simple/main.go`.

## Observações operacionais

- Em produção, prefira manter `START_LOCAL_BACKENDS=false` e usar serviços dedicados para backends.
- `ip_hash` depende do endereço remoto ou do header. Quando há proxies na frente, defina `IP_HASH_HEADER` adequadamente.
- `BACKEND_RATE_LIMITS` é posicional — se preferir configuração por URL (map), posso adicionar suporte para `URL=RPS/BURST` no futuro.

