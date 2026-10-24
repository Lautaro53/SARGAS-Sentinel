
# syntax=docker/dockerfile:1

# 1. Compilar Svelte desde el proyecto frontend.
FROM node:24-alpine AS web-build

WORKDIR /web

COPY --from=frontend package.json package-lock.json ./
RUN npm ci

COPY --from=frontend . .
RUN npm run check && npm run build

# 2. Compilar el backend Go e integrar el frontend.
FROM golang:1.27.3-alpine AS go-build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Copiamos el frontend generado por la etapa anterior.
COPY --from=web-build /web/dist ./cmd/sentinel/web/dist

RUN CGO_ENABLED=0 GOOS=linux \
    go build -trimpath -o /sentinel ./cmd/sentinel

# 3. Imagen final de ejecución.
FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=go-build /sentinel /app/sentinel

RUN addgroup -S sentinel && \
    adduser -S -G sentinel sentinel

USER sentinel

EXPOSE 8080

ENTRYPOINT ["/app/sentinel"]
