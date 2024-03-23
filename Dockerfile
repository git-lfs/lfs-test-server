FROM golang:1.20
LABEL org.opencontainers.image.authors=" GitHub, Inc."

WORKDIR /go/src/github.com/git-lfs/lfs-test-server

COPY . .

RUN go build

EXPOSE 8080

CMD /go/src/github.com/git-lfs/lfs-test-server/lfs-test-server
