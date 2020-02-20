FROM alpine:3.10.1

ADD bin/linux-amd64/ads /ads
RUN chmod +x /ads
