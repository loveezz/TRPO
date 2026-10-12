FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/hr-select ./cmd/server

FROM alpine:3.20
WORKDIR /app
RUN adduser -D -H appuser
COPY --from=build /out/hr-select /app/hr-select
COPY web /app/web
USER appuser
EXPOSE 8000
CMD ["/app/hr-select"]
