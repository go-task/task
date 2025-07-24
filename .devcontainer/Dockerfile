
FROM mcr.microsoft.com/devcontainers/universal:2 as go_builder
RUN GOBIN=/usr/local/bin go install github.com/rogpeppe/go-internal/cmd/testscript@latest
RUN GOBIN=/usr/local/bin go install github.com/go-task/task/v3/cmd/task@latest
RUN GOBIN=/usr/local/bin go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

FROM mcr.microsoft.com/devcontainers/universal:2
ENV DEBIAN_FRONTEND=noninteractive
RUN set -eux; \
    wget https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64 -O /usr/bin/yq; \
    chmod +x /usr/bin/yq;
COPY --from=go_builder /usr/local/bin/testscript /usr/local/bin/testscript
COPY --from=go_builder /usr/local/bin/task /usr/local/bin/task
COPY --from=go_builder /usr/local/bin/golangci-lint /usr/local/bin/golangci-lint
