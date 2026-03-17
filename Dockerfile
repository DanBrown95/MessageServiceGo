FROM golang:1.24 AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /messageservice .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=build /messageservice ./
COPY config/ ./config/

ARG env=development
ENV ENV=$env

ENTRYPOINT ["./messageservice"]
