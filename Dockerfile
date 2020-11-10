FROM golang:1.15.4-alpine AS build
LABEL stage=intermediate

ADD . /acfunlive-src
WORKDIR /acfunlive-src

ENV GO111MODULE=on \
    GOPROXY="https://goproxy.cn" \
    CGO_ENABLED=0
RUN go build

FROM alpine

ENV DISTFILE="/acfunlive/acfunlive" \
    CONFIGFILE="/acfunlive/config.json" \
    LIVEFILE="/acfunlive/live.json" \
    RECORD="/acfunlive/record"

RUN mkdir -p $RECORD && \
    apk update && \
    apk upgrade && \
    apk --no-cache add ffmpeg libc6-compat tzdata && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

COPY --from=build /acfunlive-src/acfunlive /acfunlive/acfunlive

VOLUME $CONFIGFILE $LIVEFILE $RECORD

ENTRYPOINT ["/acfunlive/acfunlive"]
