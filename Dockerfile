FROM golang:1.23

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

WORKDIR /app/cmd/app

RUN go build -o /go/app

EXPOSE 3000

CMD ["/go/app"]
