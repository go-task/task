FROM gcr.io/distroless/static-debian12:nonroot
COPY task /task
CMD ["/task"]
