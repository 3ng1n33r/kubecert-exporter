FROM golang:1.26 AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /app
COPY go.mod go.sum main.go ./
RUN go mod download

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o exporter .

FROM gcr.io/distroless/static:nonroot

WORKDIR /
COPY --from=builder /app/exporter .

USER nonroot:nonroot

EXPOSE 9100
ENTRYPOINT ["/exporter"]