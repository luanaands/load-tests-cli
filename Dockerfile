FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /cmd/load-tests-cli .

FROM scratch
COPY --from=build /cmd/load-tests-cli /load-tests-cli
ENTRYPOINT ["/load-tests-cli"]