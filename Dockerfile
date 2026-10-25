FROM node:24.22.1-alpine AS frontend
WORKDIR /src/web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26.10-alpine AS backend
ENV GOTOOLCHAIN=local CGO_ENABLED=0
WORKDIR /src
COPY go.mod ./
COPY . ./
COPY --from=frontend /src/internal/webassets/dist ./internal/webassets/dist
RUN GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /qoauth ./cmd/qoauth
RUN mkdir -m 0700 /state-template

FROM scratch
COPY --from=backend /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=backend /qoauth /qoauth
COPY --from=backend --chown=65532:65532 --chmod=0700 /state-template /data
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/qoauth"]
CMD ["serve", "--data-dir", "/data/state", "--listen", "0.0.0.0:8080"]
