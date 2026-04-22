FROM golang:1.26 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/two-pass-server ./cmd/two-pass-server

FROM gcr.io/distroless/static-debian12

COPY --from=build /out/two-pass-server /two-pass-server

EXPOSE 8080

ENTRYPOINT ["/two-pass-server"]
