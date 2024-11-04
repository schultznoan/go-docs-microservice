FROM golang:latest as build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./ ./

RUN ls -la /app
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./cmd/main.go

FROM alpine:latest
COPY --from=build /app/main /app/main
COPY --from=build /app/configs/config.yaml /app/config.yaml

WORKDIR /app

CMD ["./main"]
