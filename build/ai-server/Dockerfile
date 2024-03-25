FROM docker.io/library/golang:1.20 as builder

ARG GOPROXY=direct
WORKDIR /workspace

# Copy the Go Modules manifests
COPY .git .git
COPY go.mod go.mod
COPY go.sum go.sum
COPY Makefile Makefile
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN GOPROXY=${GOPROXY} go mod download

# Copy the go source
COPY cmd/ cmd/
COPY pkg/ pkg/
COPY docs/swagger-ui/ docs/swagger-ui/

# Build
RUN make ai-server

FROM alpine:3.18.4

RUN mkdir -p /opt/ai-server
WORKDIR /opt/ai-server
COPY --from=builder /workspace/bin/ai-server .
COPY --from=builder /workspace/docs/swagger-ui/ docs/swagger-ui/

EXPOSE 8080

CMD ["./ai-server"]
