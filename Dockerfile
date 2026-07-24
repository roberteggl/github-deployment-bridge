# SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
#
# SPDX-License-Identifier: Apache-2.0

FROM --platform=$BUILDPLATFORM golang:1.26-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH
ARG VERSION=dev
ARG COMMIT=unknown

RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
    -o /out/bridge ./cmd/bridge

FROM gcr.io/distroless/static-debian12:nonroot

ARG VERSION=dev
ARG COMMIT=unknown

LABEL org.opencontainers.image.title="github-deployment-bridge" \
      org.opencontainers.image.description="Bridge FluxCD reconciliations to GitHub Deployments" \
      org.opencontainers.image.source="https://github.com/roberteggl/github-deployment-bridge" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT}"

COPY --from=builder /out/bridge /bridge

USER nonroot:nonroot
EXPOSE 8080 8081
ENTRYPOINT ["/bridge"]
