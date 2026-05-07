FROM golang:1.26.3-alpine3.23@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d AS builder

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
FROM scratch

WORKDIR /

COPY --from=builder /workspace/runner .

ENTRYPOINT ["/runner"]
