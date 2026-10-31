# Imagem do Sistema de Assistência Técnica.
# Três estágios: interface (Node) → servidor (Go) → imagem final (Alpine, ~25 MB).
# A imagem só é gerada se todos os testes passarem.

FROM node:22-alpine AS interface
WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY frontend/ ./
# Grava a interface compilada em /src/internal/estatico/app
RUN npm run build

FROM golang:1.26-alpine AS servidor
ARG VERSAO=desenvolvimento
ENV CGO_ENABLED=0 GOTOOLCHAIN=local
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY --from=interface /src/internal/estatico/app ./internal/estatico/app
RUN go vet ./... \
 && go test ./... \
 && go build -trimpath -ldflags="-s -w -X main.versao=${VERSAO}" -o /out/assistencia ./cmd/assistencia

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata
COPY --from=servidor /out/assistencia /app/assistencia
ENV DIR_DADOS=/dados \
    PORTA=8089 \
    TZ=America/Sao_Paulo \
    PUID=1000 \
    PGID=1000
EXPOSE 8089
VOLUME ["/dados"]
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 CMD ["/app/assistencia", "healthcheck"]
# Começa como root só para acertar o dono da pasta de dados e logo passa a
# rodar como PUID:PGID (ver cmd/assistencia/privilegios_linux.go).
ENTRYPOINT ["/app/assistencia"]
