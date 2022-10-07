# syntax=docker/dockerfile:1

##
## Build
##
FROM golang:1.19-alpine AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /reddit-notifier

##
## Deploy
##
FROM alpine:3.16

COPY --from=build /reddit-notifier /reddit-notifier

CMD ["/reddit-notifier"]
