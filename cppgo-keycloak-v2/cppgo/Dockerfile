FROM harbor.taas.vodafone.local/baseimage/golang:1.20.2

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o main .

EXPOSE 8080

CMD [ "./main" ]
