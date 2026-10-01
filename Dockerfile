FROM golang:1.26-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 go build -mod=vendor -o /server ./cmd/server

FROM gcr.io/distroless/static-debian12
COPY --from=build /server /server
EXPOSE 8080
ENTRYPOINT ["/server"]