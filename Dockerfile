FROM golang:1.23.5-alpine3.20@sha256:def59a601e724ddac5139d447e8e9f7d0aeec25db287a9ee1615134bcda266e2 as builder

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
