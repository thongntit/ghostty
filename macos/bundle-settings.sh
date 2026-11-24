#!/bin/sh
set -eu

if [ "$#" -ne 1 ]; then
    echo "Usage: sh macos/bundle-settings.sh APP_BUNDLE" >&2
    exit 2
fi

app=$(cd "$1" && pwd)
source=$(cd "$(dirname "$0")/../settings" && pwd)
ghostty="$app/Contents/MacOS/ghostty"
helper="$app/Contents/Helpers/ghostty-config-tui"

codesign --verify --deep --strict "$app"
if ! codesign --display --verbose=4 "$app" 2>&1 | grep -qx 'Signature=adhoc'; then
    echo "Settings bundling expects an ad-hoc signed fork build." >&2
    exit 1
fi

case "$(lipo -archs "$ghostty")" in
    arm64) arch=arm64 ;;
    x86_64) arch=amd64 ;;
    *) echo "Settings bundling expects a single-architecture app." >&2; exit 1 ;;
esac

temp=$(mktemp -d)
trap 'rm -rf "$temp"' EXIT
# Generate against the executable being packaged. Build in a copy so a
# packaging run never overwrites the repository's standalone test catalog.
mkdir "$temp/settings"
cp -R "$source/." "$temp/settings/"
(
    cd "$temp/settings"
    LLVM_PROFILE_FILE="$temp/ghostty-%p.profraw" go run -mod=readonly ./cmd/schema-gen \
        --ghostty "$ghostty" --out internal/schema/options.json
    go test -mod=readonly ./...
    GOOS=darwin GOARCH="$arch" CGO_ENABLED=0 go build -mod=readonly -trimpath -o "$temp/ghostty-config-tui" ./cmd/ghostty-config-tui
)
codesign --force --sign - "$temp/ghostty-config-tui"
mkdir -p "$(dirname "$helper")"
install -m 755 "$temp/ghostty-config-tui" "$helper"
codesign --force --sign - --preserve-metadata=entitlements,flags,identifier "$app"
codesign --verify --deep --strict "$app"
