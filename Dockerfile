FROM scratch
COPY task /usr/local/bin/task
COPY completion/* /etc/completion/
CMD ["/usr/local/bin/task"]
