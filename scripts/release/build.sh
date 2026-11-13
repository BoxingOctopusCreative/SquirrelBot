#!/bin/sh
# This Source Code Form is subject to the terms of the Mozilla Public
# License, v. 2.0. If a copy of the MPL was not distributed with this
# file, You can obtain one at https://mozilla.org/MPL/2.0/.

# Builds the dashboard, then cross-compiles squirrelbot VERSION into OUTDIR
# as one .tar.gz per platform. Each archive contains squirrelbot, web/build,
# and LICENSE. VERSION is vMAJOR.MINOR.PATCH or MAJOR.MINOR.PATCH.
#
#   scripts/release/build.sh v0.1.0 dist
set -eu

version=${1:?usage: build.sh VERSION OUTDIR}
out=${2:?usage: build.sh VERSION OUTDIR}
version=${version#v}
platforms=${PLATFORMS:-linux/amd64 linux/arm64 darwin/amd64 darwin/arm64}

root=$(cd "$(dirname "$0")/../.." && pwd)
mkdir -p "$out"
out=$(cd "$out" && pwd)
stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT

cd "$root"
npm --prefix web ci
npm --prefix web run build

for platform in $platforms; do
	os=${platform%/*}
	arch=${platform#*/}
	name="squirrelbot_${version}_${os}_${arch}"
	dir="$stage/$name"
	echo "building $name"
	mkdir -p "$dir/web"
	CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath -ldflags="-s -w" \
		-o "$dir/squirrelbot" ./cmd/squirrelbot
	cp -a web/build "$dir/web/build"
	cp LICENSE "$dir/LICENSE"
	tar -C "$dir" -czf "$out/$name.tar.gz" squirrelbot web LICENSE
done
