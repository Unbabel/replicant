FROM golang:alpine AS builder
RUN apk --no-cache add git make
COPY . /src/replicant
WORKDIR /src/replicant

RUN make build

FROM alpine
COPY --from=builder /src/replicant/replicant /app/
CMD ["/app/replicant"]
