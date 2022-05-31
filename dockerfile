# syntax=docker/dockerfile:1

FROM golang:1.16-alpine AS builder
ENV CGO_ENABLED 0

WORKDIR /usr/local/go/src/mqtt-cloud-connector

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY main.go ./
COPY ./config/ /usr/local/go/src/mqtt-cloud-connector/config
COPY ./mqttbuffer/ /usr/local/go/src/mqtt-cloud-connector/mqttbuffer
COPY ./certs/ /usr/local/go/src/mqtt-cloud-connector/certs

RUN ls -laR ./

#RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOFLAGS=-mod=mod go build -ldflags="-w -s" -o /App
RUN go build -gcflags "all=-N -l" -o /App

#Step 2 - Build a small image

FROM scratch


COPY --from=builder /App /App
COPY --from=builder /usr/local/go/src/mqtt-cloud-connector/config/ /config
COPY --from=builder /usr/local/go/src/mqtt-cloud-connector/certs/ /certs

EXPOSE 1883
EXPOSE 8883

CMD [ "/App" ]

