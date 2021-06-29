FROM golang:1.16 as builder
WORKDIR /app
COPY . /app
RUN make build

FROM debian:stable-slim
COPY --from=builder /app/prom-count /prom-count
ENTRYPOINT ["/prom-count"]
