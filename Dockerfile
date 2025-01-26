FROM golang:1.23.2-alpine

COPY . /app

WORKDIR /app

RUN /app/install-task.sh -b /usr/local/bin

ENTRYPOINT ["task"]