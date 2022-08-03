FROM golang:alpine as builder

WORKDIR /app/build

COPY . .

RUN go build

FROM alpine

WORKDIR /app

COPY --from=builder /app/build /app

CMD [ "./mumble-gateway-service" ]