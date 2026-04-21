FROM golang:1.25.5-alpine AS build

WORKDIR /src

ARG TARGETOS=linux
ARG TARGETARCH=amd64

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /out/relay-api ./cmd/relay-api

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=build /out/relay-api /app/relay-api

EXPOSE 8080

ENTRYPOINT ["/app/relay-api"]
