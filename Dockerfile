FROM golang:alpine

RUN apk update && apk add gcc musl-dev upx ca-certificates dep git

WORKDIR /go/src/github.com/dirk1492/http-backend

COPY * ./

RUN dep ensure
RUN go build -ldflags "-linkmode external -extldflags -static -s -w -X main.Version=$(git rev-parse --short HEAD) -X main.Tag=$(git tag --points-at HEAD)" -o /service
RUN upx /service

FROM scratch
COPY --from=0 /service /service
COPY passwd /etc/passwd
USER nobody
ENV PATH=/
ENTRYPOINT ["service"]
