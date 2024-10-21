FROM debian:bookworm-slim
USER 65532:65532
COPY task /usr/local/bin/task
COPY completion/bash/task.bash /etc/bash_completion
WORKDIR /workspace
CMD ["/usr/local/bin/task"]
