# Imagen base agnóstica; ajustar cuando se elija el stack tecnológico.
FROM alpine:3.20 AS base
WORKDIR /app

FROM base AS development
COPY . .

FROM base AS production
COPY src ./src
