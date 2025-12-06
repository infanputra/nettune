#!/bin/bash
set -e

# Build script for nettune

VERSION=${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo "dev")}
GIT_COMMIT=${GIT_COMMIT:-$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")}
BUILD_DATE=${BUILD_DATE:-$(date -u +"%Y-%m-%dT%H:%M:%SZ")}
OUTPUT_DIR="${OUTPUT_DIR:-dist}"

LDFLAGS="-X github.com/jtsang4/nettune/pkg/version.Version=${VERSION}"
LDFLAGS="${LDFLAGS} -X github.com/jtsang4/nettune/pkg/version.GitCommit=${GIT_COMMIT}"
LDFLAGS="${LDFLAGS} -X github.com/jtsang4/nettune/pkg/version.BuildDate=${BUILD_DATE}"

echo "Building nettune ${VERSION} (${GIT_COMMIT})"
echo "Build date: ${BUILD_DATE}"
echo ""

mkdir -p "${OUTPUT_DIR}"

# Build for multiple platforms
PLATFORMS=(
  "darwin/amd64"
  "darwin/arm64"
  "linux/amd64"
  "linux/arm64"
)

for platform in "${PLATFORMS[@]}"; do
  IFS="/" read -r GOOS GOARCH <<< "${platform}"
  output="${OUTPUT_DIR}/nettune-${GOOS}-${GOARCH}"

  echo "Building for ${GOOS}/${GOARCH}..."
  GOOS=${GOOS} GOARCH=${GOARCH} go build -ldflags "${LDFLAGS}" -o "${output}" ./cmd/nettune

  if [ $? -eq 0 ]; then
    echo "  -> ${output}"
  else
    echo "  -> Failed!"
    exit 1
  fi
done

# Generate checksums
echo ""
echo "Generating checksums..."
cd "${OUTPUT_DIR}"
sha256sum nettune-* > checksums.txt
echo "  -> checksums.txt"

echo ""
echo "Build complete!"
ls -la
