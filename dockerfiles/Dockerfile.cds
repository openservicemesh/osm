FROM alpine:3.10.1

ADD bin/linux-amd64/cds /cds
RUN chmod +x /cds
