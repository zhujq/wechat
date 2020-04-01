FROM alpine:latest

ADD entrypoint.sh /entrypoint.sh
ADD mywechat /mywechat
ADD wechat-index /wechat-index
ADD wechat-db /wechat-db
RUN  chmod +x /mywechat && chmod +x /wechat-index && chmod +x /wechat-db  && chmod 777 /entrypoint.sh
ENTRYPOINT  /entrypoint.sh 

EXPOSE 80
