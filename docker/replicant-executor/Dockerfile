FROM golang:alpine AS builder
RUN apk --no-cache add git make
COPY . /src/replicant
WORKDIR /src/replicant

RUN make build

FROM chromedp/headless-shell:stable
COPY --from=builder /src/replicant/replicant-cdp /app/
ENTRYPOINT []
CMD ["/app/replicant-cdp"]
