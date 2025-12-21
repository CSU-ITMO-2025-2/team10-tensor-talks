FROM golang:1.24-alpine AS build
WORKDIR /app
COPY go.mod ./
COPY go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o mock-interview-builder-service ./cmd/mock-interview-builder-service

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /app/mock-interview-builder-service /usr/local/bin/mock-interview-builder-service
COPY config ./config
EXPOSE 8089
ENTRYPOINT ["/usr/local/bin/mock-interview-builder-service"]

