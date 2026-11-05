#!/usr/bin/env bash
# Build locally and export a self-contained release for a Linux AMD64 server.
set -euo pipefail

if [[ $# != 1 || ! $1 =~ ^[a-zA-Z0-9_][a-zA-Z0-9_.-]{0,127}$ ]]; then
  echo "Usage: $0 RELEASE_TAG (a unique Docker tag, e.g. 2026-11-04.1)" >&2
  exit 2
fi
export RELEASE_TAG=$1
repo_root=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
cd "$repo_root"
for command_name in docker python3 tar gzip sha256sum; do
  command -v "$command_name" >/dev/null || { echo "Missing command: $command_name" >&2; exit 1; }
done
[[ -f deploy/build.env ]] || { echo 'Create deploy/build.env from deploy/build.env.example first.' >&2; exit 1; }
[[ -f deploy/secrets/sentry_auth_token ]] || { echo 'Create deploy/secrets/sentry_auth_token (empty if unused).' >&2; exit 1; }
release_dir="$repo_root/deploy/releases/$RELEASE_TAG"
[[ ! -e $release_dir ]] || { echo "Release already exists: $release_dir" >&2; exit 1; }
mkdir -p deploy/releases
stage_dir=$(mktemp -d "$repo_root/deploy/releases/.package.XXXXXX")
trap 'rm -rf -- "$stage_dir"' EXIT
build_compose=(docker compose --env-file deploy/build.env -f compose.build.yml)

# Export only allowlisted public build values, never local env files or secrets.
"${build_compose[@]}" config --format json > "$stage_dir/build.json"
python3 - "$stage_dir" <<'PY'
import json, os, pathlib, sys
from urllib.parse import urlsplit
stage = pathlib.Path(sys.argv[1])
services = json.loads((stage / 'build.json').read_text())['services']
web = services['website']['build']['args']
admin = services['admin']['build']['args']
values = {'RELEASE_TAG': os.environ['RELEASE_TAG'],
          'SITE_ORIGIN': web['CUSTOMER_PUBLIC_ORIGIN'],
          'ADMIN_ORIGIN': admin['VITE_API_URL'].removesuffix('/api/v1')}
for key in ('SITE_ORIGIN', 'ADMIN_ORIGIN'):
    value = values[key]
    url = urlsplit(value)
    if url.scheme != 'https' or not url.hostname or url.path or url.query or url.fragment or url.username or url.password:
        raise SystemExit(f'{key} must be an HTTPS origin without a path or credentials')
for key in ('MENU_IMAGE_HOST', 'NEXT_PUBLIC_SENTRY_DSN', 'NEXT_PUBLIC_SENTRY_RELEASE', 'NEXT_PUBLIC_SENTRY_TRACES_SAMPLE_RATE'):
    values[key] = web.get(key, '')
bundle = stage / 'bundle'
(bundle / 'deploy/nginx').mkdir(parents=True)
text = pathlib.Path('deploy/production.env.example').read_text()
lines = []
for line in text.splitlines():
    key = line.split('=', 1)[0]
    if key in values:
        value = str(values[key])
        if any(c in value for c in "\r\n'"):
            raise SystemExit(f'Unsupported character in {key}')
        line = f"{key}='{value}'"
    lines.append(line)
(bundle / 'deploy/production.env.example').write_text('\n'.join(lines) + '\n')
PY
"${build_compose[@]}" build --pull
docker pull --platform linux/amd64 postgres:17
docker tag postgres:17 "citypark/postgres:$RELEASE_TAG"
images=()
for service in backend website admin postgres; do
  image="citypark/$service:$RELEASE_TAG"
  platform=$(docker image inspect --format '{{.Os}}/{{.Architecture}}' "$image")
  [[ $platform == linux/amd64 ]] || { echo "Unexpected platform for $image: $platform" >&2; exit 1; }
  images+=("$image")
  docker image inspect --format '{{join .RepoTags ","}} {{.Id}} {{.Os}}/{{.Architecture}}' "$image" >> "$stage_dir/bundle/IMAGES.txt"
done
docker image save "${images[@]}" | gzip > "$stage_dir/images.tar.gz"
cp compose.production.yml "$stage_dir/bundle/"
cp deploy/README.md deploy/backend.env.example deploy/website.env.example "$stage_dir/bundle/deploy/"
cp deploy/nginx/citypark.conf.example "$stage_dir/bundle/deploy/nginx/"
printf '%s\n' "$RELEASE_TAG" > "$stage_dir/bundle/RELEASE"
tar -czf "$stage_dir/deployment.tar.gz" -C "$stage_dir/bundle" .
(cd "$stage_dir" && sha256sum images.tar.gz deployment.tar.gz > SHA256SUMS)
rm -rf -- "$stage_dir/bundle" "$stage_dir/build.json"
mv -- "$stage_dir" "$release_dir"
printf 'Release ready: %s\nTransfer images.tar.gz, deployment.tar.gz and SHA256SUMS.\n' "$release_dir"
