FROM docker.io/apache/skywalking-go:0.2.0-go1.20 as sky
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

# Build
RUN make ai-controller-manager

FROM docker.io/library/alpine:3.18.4
WORKDIR /
COPY --from=builder /workspace/bin/ai-controller-manager .
USER 65532:65532

ENTRYPOINT ["/ai-controller-manager"]
