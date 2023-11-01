FROM alpine

RUN apk add --no-cache wireguard-tools bash wget openresolv iptables

WORKDIR /

COPY cmd/bin/octopus .

ENTRYPOINT ["./octopus"]
