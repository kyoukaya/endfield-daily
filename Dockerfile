FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /endfield-daily .

FROM alpine:3.21
COPY --from=build /endfield-daily /endfield-daily
ENTRYPOINT ["/endfield-daily"]
