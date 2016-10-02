FROM gliderlabs/alpine

ADD main /

ENTRYPOINT /main

EXPOSE 8081
