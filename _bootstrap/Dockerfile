FROM golang

RUN go get github.com/h2so5/murcott

WORKDIR /go/src/github.com/h2so5/murcott/_bootstrap

RUN go get . ...

RUN go build -o murcott-bt main.go
