FROM gcr.io/distroless/static-debian12:nonroot
COPY task /usr/local/bin/task
CMD ["/usr/local/bin/task"]
