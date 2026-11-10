FROM golang:1.24 AS development
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
EXPOSE 8080
CMD ["go", "run", "./src/cmd/server"]

FROM development AS build
RUN CGO_ENABLED=0 go build -o /server ./src/cmd/server

FROM alpine:3.20 AS production
WORKDIR /app
COPY --from=build /server /app/server
RUN mkdir -p /app/data && chown -R 10001:10001 /app/data
USER 10001:10001
EXPOSE 8080
CMD ["/app/server"]
