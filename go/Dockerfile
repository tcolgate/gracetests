# build stage
FROM golang:1.11 AS build-env
ADD . /src
RUN cd /src &&  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o goapp .

# final stage
FROM scratch
WORKDIR /app
COPY --from=build-env /src/goapp /app/
ENTRYPOINT ["/app/goapp"]
