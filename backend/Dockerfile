FROM golang:1.27.4-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/vms-api ./cmd \
    && CGO_ENABLED=0 go build -trimpath -o /out/vms-migrate ./cmd/vms-migrate

# Optional local target uses host-built binaries to avoid a compiler-image download.
FROM gcr.io/distroless/static-debian12:nonroot AS local
WORKDIR /app
COPY --from=vms_binaries /vms-api /vms-api
COPY --from=vms_binaries /vms-migrate /vms-migrate
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/vms-api"]

FROM builder AS dev
ENV VMS_LISTEN=0.0.0.0:8080
EXPOSE 8080
CMD ["go", "run", "./cmd"]

FROM gcr.io/distroless/static-debian12:nonroot AS prod
WORKDIR /app
COPY --from=builder /out/vms-api /vms-api
COPY --from=builder /out/vms-migrate /vms-migrate
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/vms-api"]
