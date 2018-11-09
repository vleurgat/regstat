FROM golang:alpine AS build
RUN apk add git
ENV GOPATH=/go
ADD . /go/src/github.com/vleurgat/regstat
WORKDIR /go/src/github.com/vleurgat/regstat
RUN go get -d ./...
RUN go install github.com/vleurgat/regstat/cmd/regstat

FROM alpine
WORKDIR /usr/local/bin
COPY --from=build /go/bin/regstat /usr/local/bin
RUN printf '#!/bin/sh\nexec /usr/local/bin/regstat -pg-conn-str "host=localhost port=5432 user=postgres sslmode=disable" "$@"\n' > /usr/local/bin/entrypoint.sh && chmod a+x /usr/local/bin/entrypoint.sh
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
EXPOSE 3333