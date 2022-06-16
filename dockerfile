# syntax=docker/dockerfile:1

FROM golang:1.16-alpine AS builder
ENV CGO_ENABLED 0

WORKDIR /usr/local/go/src/opcua-mqtt-connector

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY main.go ./
COPY ./config/ /usr/local/go/src/opcua-mqtt-connector/config
#COPY ./certs/ /usr/local/go/src/opcua-mqtt-connector/certs

RUN ls -laR ./

#RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GOFLAGS=-mod=mod go build -ldflags="-w -s" -o /App
RUN go build -gcflags "all=-N -l" -o /OpcuaMqttApp

#Step 2 - Build a small image

FROM scratch


COPY --from=builder /OpcuaMqttApp /OpcuaMqttApp
COPY --from=builder /usr/local/go/src/opcua-mqtt-connector/config/ /config
#COPY --from=builder /usr/local/go/src/opcua-mqtt-connector/certs/ /certs

#EXPOSE 1883
#EXPOSE 8883

CMD [ "/OpcuaMqttApp" ]

