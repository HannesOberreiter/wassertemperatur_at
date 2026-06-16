FROM golang:1.25-bookworm AS build
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . ./
RUN CGO_ENABLED=0 go build -o /wassertemperatur

FROM gcr.io/distroless/static-debian12
WORKDIR /
COPY --from=build /wassertemperatur /wassertemperatur
COPY --from=build /app/assets /assets
VOLUME ["/db"]
EXPOSE 1323
ENV SQL_PATH=/db/wasser.db
CMD ["/wassertemperatur"]
