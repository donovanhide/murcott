FROM golang

RUN go get github.com/h2so5/murcott

WORKDIR /go/src/github.com/h2so5/murcott/_gui

RUN go get . ...

RUN go build -o murcott-gui main.go

ENTRYPOINT ./murcott-gui

EXPOSE 3000
