FROM alpine:3.10.1

ADD bin/linux-amd64/lds /lds
RUN chmod +x /lds
