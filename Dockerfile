# Build the runner binary
FROM golang:1.17 as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY src/ src/

# Build
RUN CGO_ENABLED=0 GO111MODULE=on go build -a -o runner main.go

# Use distroless as minimal base image to package the runner binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM quay.io/prometheus/node-exporter:master
USER root
RUN mkdir -p /prometheus/node_exporter/textfile_collector
ADD buffer-size.sh /prometheus/buffer-size.sh
RUN chmod 0744 /prometheus/buffer-size.sh
WORKDIR /
COPY --from=builder /workspace/runner .
ENTRYPOINT ["/runner"]
