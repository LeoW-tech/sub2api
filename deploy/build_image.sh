#!/usr/bin/env bash
# 本地构建镜像的快速脚本，避免在命令行反复输入构建参数。

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
NPM_REGISTRY="${NPM_REGISTRY:-https://registry.npmjs.org}"
COMMIT_SHA="${COMMIT_SHA:-$(git -C "${REPO_ROOT}" rev-parse HEAD)}"
BUILD_DATE="${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

if [[ ! "${COMMIT_SHA}" =~ ^[0-9a-f]{40}$ ]]; then
    echo "COMMIT_SHA must be a 40-character Git commit SHA" >&2
    exit 1
fi

docker build \
    -t "sub2api:${COMMIT_SHA}" \
    -t sub2api:latest \
    --label "org.opencontainers.image.revision=${COMMIT_SHA}" \
    --build-arg GOPROXY=https://goproxy.cn,direct \
    --build-arg GOSUMDB=sum.golang.google.cn \
    --build-arg NPM_REGISTRY="${NPM_REGISTRY}" \
    --build-arg COMMIT="${COMMIT_SHA}" \
    --build-arg DATE="${BUILD_DATE}" \
    -f "${REPO_ROOT}/Dockerfile" \
    "${REPO_ROOT}"
