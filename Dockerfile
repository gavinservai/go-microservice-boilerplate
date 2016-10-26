FROM gliderlabs/alpine

ADD main /

RUN apk-install ca-certificates

ENTRYPOINT /main

EXPOSE 8081
