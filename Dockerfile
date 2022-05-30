FROM golang:1.17-alpine AS build

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . . 

RUN go build 

FROM alpine

COPY --from=build /app/dvc-http-remote /bin

WORKDIR /app
EXPOSE 8080

ENTRYPOINT ["dvc-http-remote"]
