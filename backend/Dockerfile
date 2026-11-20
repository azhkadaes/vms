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
COPY --chmod=755 --from=vms_binaries /vms-api /vms-api
COPY --chmod=755 --from=vms_binaries /vms-migrate /vms-migrate
COPY --chown=65532:65532 ops/stage1/upload-root/ /var/lib/vms/uploads/
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/vms-api"]

FROM builder AS dev
ENV APP_HOST=0.0.0.0 APP_PORT=8080
EXPOSE 8080
CMD ["go", "run", "./cmd"]

FROM gcr.io/distroless/static-debian12:nonroot AS prod
WORKDIR /app
COPY --from=builder /out/vms-api /vms-api
COPY --from=builder /out/vms-migrate /vms-migrate
COPY --chown=65532:65532 ops/stage1/upload-root/ /var/lib/vms/uploads/
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/vms-api"]
