#!/usr/bin/env sh
set -eu

VERSION="${VERSION:-dev}"
OUT_DIR="${OUT_DIR:-dist}"
APP="chasqui-local-agent"
PLATFORMS="${PLATFORMS:-linux/amd64 linux/arm64 windows/amd64 darwin/amd64 darwin/arm64}"

rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"

for platform in $PLATFORMS; do
  GOOS="${platform%/*}"
  GOARCH="${platform#*/}"
  EXT=""
  if [ "$GOOS" = "windows" ]; then
    EXT=".exe"
  fi
  NAME="$APP-$VERSION-$GOOS-$GOARCH$EXT"
  echo "building $NAME"
  CGO_ENABLED=0 GOOS="$GOOS" GOARCH="$GOARCH" go build \
    -trimpath \
    -ldflags "-s -w -X main.version=$VERSION" \
    -o "$OUT_DIR/$NAME" .
done

(
  cd "$OUT_DIR"
  sha256sum * > SHA256SUMS.txt
)

echo "release artifacts written to $OUT_DIR"
