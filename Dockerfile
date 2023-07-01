FROM golang:1.20 AS builder

ENV GO111MODULE=on \
  CGO_ENABLED=0 \
  GOOS=linux \
  GOARCH=amd64

WORKDIR /app

COPY . .

RUN go build \
  -trimpath \
  -ldflags "-s -w -X main.BuildTag=$(git describe --tags --abbrev=0) -X main.BuildName=golurectl -extldflags '-static'" \
  -o /bin/golurectl \
  ./cmd/golurectl

FROM scratch
COPY --from=builder /bin/golurectl /bin/golurectl

ENTRYPOINT ["/bin/golurectl"]