FROM golang:1.22 AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/api-server ./cmd/api-server
RUN CGO_ENABLED=0 go build -trimpath -o /out/worker ./cmd/worker
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api-server /api-server
COPY --from=build /out/worker /worker
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/api-server"]
