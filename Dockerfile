FROM golang:1.19
WORKDIR /go/src/booking-webapp
COPY . .
RUN go build -o bin/server ./main.go
CMD ["./bin/server"]