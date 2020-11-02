FROM golang:1.14 AS builder
ENV GOPROXY https://goproxy.io
ENV CGO_ENABLED 0
WORKDIR /go/src/app
ADD . .
RUN go build -mod vendor -o /enforce-qcloud-internal-lb

FROM alpine:3.12
COPY --from=builder /enforce-qcloud-internal-lb /enforce-qcloud-internal-lb
CMD ["/enforce-qcloud-internal-lb"]