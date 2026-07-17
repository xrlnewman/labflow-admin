FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY server ./server
RUN CGO_ENABLED=0 go build -o /labflow-admin-api ./server
FROM alpine:3.22
COPY --from=build /labflow-admin-api /labflow-admin-api
EXPOSE 8080
ENTRYPOINT ["/labflow-admin-api"]
