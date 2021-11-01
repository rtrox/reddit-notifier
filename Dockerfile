# syntax=docker/dockerfile:1

##
## Build
##
FROM golang:1.17-buster AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /reddit-notifier

##
## Deploy
##
FROM gcr.io/distroless/base-debian10

WORKDIR /

COPY --from=build /reddit-notifier /reddit-notifier

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/reddit-notifier"]
