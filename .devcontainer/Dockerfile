ARG GOLANG_VERSION=1.25
ARG GOLANGCI_VERSION=v2.1.0

FROM mcr.microsoft.com/devcontainers/go:${GOLANG_VERSION} AS go_builder
ARG GOLANGCI_VERSION
RUN GOBIN=/usr/local/go/bin go install github.com/go-task/task/v3/cmd/task@latest
RUN GOBIN=/usr/local/go/bin go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${GOLANGCI_VERSION}

FROM mcr.microsoft.com/devcontainers/go:${GOLANG_VERSION}
ENV DEBIAN_FRONTEND=noninteractive
COPY --from=go_builder /usr/local/go/bin/task /usr/local/go/bin/task
COPY --from=go_builder /usr/local/go/bin/golangci-lint /usr/local/go/bin/golangci-lint
