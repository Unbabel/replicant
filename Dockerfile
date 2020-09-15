FROM golang:1.15-alpine AS builder
RUN apk --no-cache add git make
COPY . /src/replicant
WORKDIR /src/replicant

RUN make build

FROM chromedp/headless-shell:stable
RUN apt update \
&& apt install -y ca-certificates \
&& apt clean; apt clean \
&& rm -rf /var/lib/apt/lists/*

COPY --from=builder /src/replicant/replicant /app/
ENTRYPOINT []
CMD ["/app/replicant"]
