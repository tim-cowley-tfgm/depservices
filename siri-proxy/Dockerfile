FROM golang:alpine AS build-env

LABEL maintainer="Chris March <Chris.March@tfgm.com>"

ARG GITHUB_ACCESS_TOKEN

RUN apk add --no-cache git

RUN git config --global url."https://${GITHUB_ACCESS_TOKEN}:@github.com/".insteadOf "https://github.com/"

ENV CGO_ENABLED 0

WORKDIR /src

ADD ./*.go ./

RUN go get -d -v ./...
RUN go build -o siri-proxy

FROM alpine

RUN apk update
RUN apk upgrade
RUN rm -rf /var/cache/apk/*

COPY --from=build-env /src/siri-proxy /app/

WORKDIR /app

ENV SIRI_PROXY_SERVER_URL=${SIRI_PROXY_SERVER_URL}
ENV SIRI_PROXY_SERVER_PORT=${SIRI_PROXY_SERVER_PORT:-8080}
ENV SIRI_PROXY_TARGET_URL=${SIRI_PROXY_TARGET_URL}
ENV SIRI_DEFAULT_HEARTBEAT_NOTIFICATION_INTERVAL=${SIRI_DEFAULT_HEARTBEAT_NOTIFICATION_INTERVAL:-"PT5M"}
ENV HTTP_CLIENT_TIMEOUT=${HTTP_CLIENT_TIMEOUT:-10}

EXPOSE ${SERVER_PORT}

CMD ["./siri-proxy"]
