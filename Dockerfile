FROM golang:1.23.6-alpine3.20@sha256:22caeb4deced0138cb4ae154db260b22d1b2ef893dde7f84415b619beae90901 as builder

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
