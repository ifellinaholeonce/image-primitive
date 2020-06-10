FROM golang:1.14
ADD . /go/src/image-primitive
WORKDIR /go/src/image-primitive
RUN go get image-primitive
RUN go get -u github.com/fogleman/primitive
RUN go install
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build  -o /main
CMD ["/main"]
EXPOSE 3000
