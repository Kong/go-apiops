FROM golang:1.20.3 AS build
WORKDIR /kced
COPY go.mod ./
COPY go.sum ./
RUN go mod download
ADD . .
ARG COMMIT
ARG TAG
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o kced \
      -ldflags "-s -w -X github.com/kong/go-apiops/cmd.VERSION=$TAG -X github.com/kong/go-apiops/cmd.COMMIT=$COMMIT"

FROM alpine:3.17.3
RUN adduser --disabled-password --gecos "" kceduser
RUN apk --no-cache add ca-certificates jq
USER kceduser
COPY --from=build /kced/kced /usr/local/bin
ENTRYPOINT ["kced"]
