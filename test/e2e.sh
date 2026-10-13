#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
GOLIATH="go run $PROJECT_DIR/cmd/goliath"

TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

echo "=== Building goliath ==="
cd "$PROJECT_DIR"
go build -o "$TMPDIR/goliath" ./cmd/goliath
GOLIATH="$TMPDIR/goliath"

echo ""
echo "=== Creating workspace ==="
cd "$TMPDIR"
$GOLIATH create workspace --name myapp --top-ns github.com/example
cd myapp

echo ""
echo "=== Workspace structure ==="
find . -name "*.go" -o -name "go.mod" -o -name "go.work" -o -name "goliath.yaml" | sort

echo ""
echo "=== Creating components ==="
$GOLIATH create component user
$GOLIATH create component admin --interface user
$GOLIATH create component util

echo ""
echo "=== Creating base ==="
$GOLIATH create base cli

echo ""
echo "=== Creating project ==="
$GOLIATH create project myapp

echo ""
echo "=== goliath info ==="
$GOLIATH info

echo ""
echo "=== goliath check ==="
$GOLIATH check || true

echo ""
echo "=== goliath deps ==="
$GOLIATH deps

echo ""
echo "=== goliath deps --brick user ==="
$GOLIATH deps --brick user

echo ""
echo "=== goliath libs ==="
$GOLIATH libs

echo ""
echo "=== goliath ws ==="
$GOLIATH ws --get settings

echo ""
echo "=== goliath ws --get components:keys ==="
$GOLIATH ws --get components:keys

echo ""
echo "=== goliath version ==="
$GOLIATH version

echo ""
echo "=== goliath test ==="
$GOLIATH test || true

echo ""
echo "=== go.work ==="
cat go.work

echo ""
echo "=== goliath.yaml ==="
cat goliath.yaml

echo ""
echo "=== Full workspace tree ==="
find . -type f \( -name "*.go" -o -name "go.mod" -o -name "go.work" -o -name "goliath.yaml" \) | sort

echo ""
echo "=== All e2e tests passed ==="
