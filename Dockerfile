FROM golang:1.16-alpine3.13 as builder

WORKDIR $GOPATH/src/wechat
COPY . .

RUN apk add --no-cache git && set -x && \
    go mod init && go get -d -v
RUN CGO_ENABLED=0 GOOS=linux go build -o /mywechat mywechat-main.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /wechat-db wechat-db.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /wechat-index wechat-index.go
RUN CGO_ENABLED=0 GOOS=linux go build -o /wechat-token wechat-token.go

FROM alpine:latest

WORKDIR /
COPY --from=builder /mywechat . 
COPY --from=builder /wechat-db . 
COPY --from=builder /wechat-index .
COPY --from=builder /wechat-token .
ADD entrypoint.sh /entrypoint.sh
ADD account.json /account.json

RUN  chmod +x /mywechat  /wechat-index /wechat-db /wechat-token && chmod 777 /entrypoint.sh
ENTRYPOINT  /entrypoint.sh 

EXPOSE 80
