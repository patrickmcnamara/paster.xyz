FROM golang

WORKDIR /go/src/paster.xyz
COPY . .

ENV GO111MODULE on

RUN go get -d ./...
RUN go install ./...

CMD ["paster.xyz"]
