FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/wallet ./cmd

FROM alpine:3.20
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/wallet /usr/local/bin/wallet
EXPOSE 8080
ENTRYPOINT ["wallet"]