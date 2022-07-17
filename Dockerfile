FROM golang:1.18.3-alpine AS build
WORKDIR /root
RUN apk --no-cache add git ca-certificates
RUN git clone --depth 1 https://github.com/fanxcv/Maven-Go.git && \
    cd Maven-Go && go mod tidy && \
    go build -o MavenGo src/main.go && \
    chmod a+x MavenGo

FROM alpine
COPY --from=build /root/Maven-Go/MavenGo /usr/local/bin/
COPY --from=build /root/Maven-Go/config.yaml /root/config.yaml

VOLUME ["/data"]

ENTRYPOINT ["MavenGo"]

CMD ["-c", "/root/config.yaml"]