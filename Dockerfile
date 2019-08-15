FROM golang:alpine

RUN apk update && apk add gcc musl-dev upx ca-certificates dep git

COPY *.go /go/src/github.com/dirk1492/default-backend/
COPY Gopkg.* /go/src/github.com/dirk1492/default-backend/

WORKDIR /go/src/github.com/dirk1492/default-backend

RUN dep ensure
RUN go build -ldflags "-linkmode external -extldflags -static -s -w" -o prog
RUN upx prog

FROM scratch
COPY --from=0 /go/src/github.com/dirk1492/default-backend /service
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
CMD ["/service"]