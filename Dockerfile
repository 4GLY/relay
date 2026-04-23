FROM --platform=$BUILDPLATFORM golang:1.25.5-alpine AS build

WORKDIR /src

ARG TARGETOS
ARG TARGETARCH

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /out/relay-api ./cmd/relay-api
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /out/relay-worker ./cmd/relay-worker

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=build /out/relay-api /app/relay-api
COPY --from=build /out/relay-worker /app/relay-worker

EXPOSE 8080

ENTRYPOINT ["/app/relay-api"]
