FROM node:alpine AS node_build
LABEL stage=buildnode

ADD acfunlive-ui /acfunlive-ui-src
WORKDIR /acfunlive-ui-src

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories && \
    apk update && \
    apk add yarn && \
    yarn config set registry "https://registry.npm.taobao.org/" && \
    yarn install && \
    yarn generate

FROM golang:1-alpine AS go_build
LABEL stage=buildgo

ADD . /acfunlive-src
WORKDIR /acfunlive-src

ENV GO111MODULE=on \
    GOPROXY="https://goproxy.cn" \
    CGO_ENABLED=0

RUN go build

FROM alpine

ENV BINFILE="/acfunlive/acfunlive" \
    WEBUIDIR="/acfunlive/webui" \
    CONFIGDIR="/acfunlive/config" \
    RECORDDIR="/acfunlive/record"

EXPOSE 51880
EXPOSE 51890

RUN mkdir -p $WEBUIDIR && \
    mkdir -p $CONFIGDIR && \
    mkdir -p $RECORDDIR && \
    sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories && \
    apk update && \
    apk upgrade && \
    apk --no-cache add ffmpeg libc6-compat tzdata && \
    ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime

COPY --from=node_build /acfunlive-ui-src/dist $WEBUIDIR
COPY --from=go_build /acfunlive-src/acfunlive $BINFILE

VOLUME $CONFIGDIR $RECORDDIR

ENTRYPOINT ["/acfunlive/acfunlive", "-config", "/acfunlive/config", "-record", "/acfunlive/record"]
