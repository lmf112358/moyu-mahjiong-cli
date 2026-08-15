#!/usr/bin/env bash
#
# moyu-mahjong-cli 一键发布脚本
#
# 功能：交叉编译全平台 x86 二进制 → 打包 →（可选）上传到 GitHub Release
#
# 用法：
#   bash scripts/release.sh              # 仅编译 + 打包到 dist/
#   GH_TOKEN=xxx bash scripts/release.sh # 编译 + 打包 + 上传到 GitHub Release
#
# 版本号自动从 cmd/majiang/main.go 的 const version 提取。
# 若根目录存在 RELEASE_NOTES.md，其内容将作为 Release 正文，否则使用默认模板。
#
set -euo pipefail

# ---------- 配置 ----------
MAIN_GO="cmd/majiang/main.go"
OWNER_REPO="LimitlessMindForce/moyu-mahjiong-cli"
DIST="dist"
APP="majiang"

# 目标矩阵：goos goarch ext
TARGETS=(
  "darwin amd64   "
  "darwin arm64   "
  "linux  amd64   "
  "linux  386     "
  "windows amd64 .exe"
  "windows 386   .exe"
)

# ---------- 依赖检查 ----------
need() { command -v "$1" >/dev/null 2>&1 || { echo "缺少依赖：$1" >&2; exit 1; }; }
need go
need tar
if [ -n "${GH_TOKEN:-}" ]; then
  need curl
  need jq
fi

# ---------- 版本号 ----------
VERSION=$(sed -nE 's/^const version = "([^"]+)".*/\1/p' "$MAIN_GO" | head -n1)
if [ -z "${VERSION:-}" ]; then
  echo "无法从 $MAIN_GO 提取版本号（const version = \"x.y.z\"）" >&2
  exit 1
fi
TAG="v$VERSION"
echo "==> 版本 $VERSION  (tag $TAG)"

# ---------- 打包单个目标 ----------
make_archive() {
  local dir="$1" archive="$2" binary="$3"
  case "$archive" in
    *.zip)
      if command -v zip >/dev/null 2>&1; then
        zip -q -j "$archive" "$dir/$binary" "$dir/README.md"
      elif command -v powershell.exe >/dev/null 2>&1; then
        local wdir warch
        wdir=$(cygpath -w "$dir" 2>/dev/null || printf '%s' "$dir")
        warch=$(cygpath -w "$archive" 2>/dev/null || printf '%s' "$archive")
        powershell.exe -NoProfile -Command \
          "Compress-Archive -Path '$wdir\\$binary','$wdir\\README.md' -DestinationPath '$warch' -Force"
      else
        echo "无 zip 也无 powershell.exe，无法打包 $archive" >&2; exit 1
      fi
      ;;
    *.tar.gz)
      tar -C "$dir" -czf "$archive" "$binary" README.md
      ;;
  esac
}

# ---------- 编译 + 打包 ----------
echo "==> 编译全平台 x86"
rm -rf "$DIST"
for t in "${TARGETS[@]}"; do
  # shellcheck disable=SC2086
  set -- $t
  goos="$1"; goarch="$2"; ext="${3:-}"
  binary="${APP}${ext}"
  name="${APP}_${VERSION}_${goos}_${goarch}"
  dir="$DIST/.build/$name"
  archive_tar="$DIST/${name}.tar.gz"
  archive_zip="$DIST/${name}.zip"
  mkdir -p "$dir"
  printf "   • %-28s" "$goos/$goarch"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go build -trimpath -ldflags="-s -w" -o "$dir/$binary" ./cmd/majiang
  cp README.md "$dir/"
  if [ "$goos" = "windows" ]; then
    make_archive "$dir" "$archive_zip" "$binary"
    echo "$(basename "$archive_zip")"
  else
    make_archive "$dir" "$archive_tar" "$binary"
    echo "$(basename "$archive_tar")"
  fi
done
rm -rf "$DIST/.build"

echo "==> 产物 ($DIST/)"
ls -lh "$DIST"/*.tar.gz "$DIST"/*.zip 2>/dev/null | awk '{print "   " $9 "  " $5}'

# ---------- 可选：上传 GitHub Release ----------
if [ -z "${GH_TOKEN:-}" ]; then
  echo "==> 未设置 GH_TOKEN，跳过上传。"
  echo "   上传：GH_TOKEN=<token> bash scripts/release.sh"
  exit 0
fi

echo "==> 上传到 GitHub Release"
API="https://api.github.com/repos/$OWNER_REPO"
AUTH=(-H "Authorization: token $GH_TOKEN" -H "Accept: application/vnd.github+json")

# 查 / 建 Release
REL_ID=$(curl -s "${AUTH[@]}" "$API/releases/tags/$TAG" | jq -r '.id // empty')
if [ -z "$REL_ID" ]; then
  if [ -f RELEASE_NOTES.md ]; then
    BODY=$(jq -n --rawfile n RELEASE_NOTES.md --arg t "$TAG" '{tag_name:$t,name:$t,body:$n,draft:false,prerelease:false}')
  else
    BODY=$(jq -n --arg t "$TAG" --arg v "$VERSION" \
      '{tag_name:$t,name:$t,body:("moyu-mahjong-cli "\($v)"\n\n全平台预编译二进制：darwin(amd64/arm64)、linux(amd64/386)、windows(amd64/386)。\n解压即用：./majiang play"),draft:false,prerelease:false}')
  fi
  REL_ID=$(printf '%s' "$BODY" | curl -s "${AUTH[@]}" -H "Content-Type: application/json" \
    --data @- -X POST "$API/releases" | jq -r '.id')
  if [ -z "$REL_ID" ]; then echo "!!! 创建 Release 失败" >&2; exit 1; fi
  echo "   已创建 Release id=$REL_ID"
else
  echo "   Release 已存在 id=$REL_ID，追加 assets"
fi

UPLOAD="https://uploads.github.com/repos/$OWNER_REPO/releases/$REL_ID/assets"
for f in "$DIST"/*.tar.gz "$DIST"/*.zip; do
  [ -f "$f" ] || continue
  name=$(basename "$f")
  case "$name" in *.zip) ct="application/zip";; *) ct="application/gzip";; esac
  printf "   • %-40s" "$name"
  curl -s "${AUTH[@]}" -H "Content-Type: $ct" --data-binary @"$f" \
    "$UPLOAD?name=$name" | jq -r '"state=" + (.state // .message)'
done

HTML_URL=$(curl -s "${AUTH[@]}" "$API/releases/$REL_ID" | jq -r .html_url)
echo "==> 完成：$HTML_URL"
