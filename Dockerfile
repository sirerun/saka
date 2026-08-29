# syntax=docker/dockerfile:1
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags "-s -w" -o /saka ./cli/saka

FROM alpine:3.24
RUN adduser -D -u 10001 saka && apk add --no-cache ca-certificates
COPY --from=build /saka /usr/local/bin/saka
USER saka
EXPOSE 8080
ENTRYPOINT ["saka"]
CMD ["serve", "--addr", ":8080"]
