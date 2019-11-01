FROM golang:alpine AS builder
RUN apk --no-cache add git
ADD . /src/replicant
WORKDIR /src/replicant
RUN CGO_ENABLED=0 go build -mod=vendor -ldflags '-w -extldflags "-static"' -o replicant cmd/replicant/*.go

FROM alpine
# FROM gcr.io/distroless/base
WORKDIR /app
COPY --from=builder /src/replicant/replicant /app/
CMD ["/app/replicant"]
