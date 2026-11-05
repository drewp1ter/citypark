# Production deployment from image archives

Build on your development machine, then transfer prebuilt Linux AMD64 images over SSH. The server needs Docker Engine, Compose v2, host nginx, and runtime configuration. No source checkout, compiler, registry login, or Sentry build token is needed on the server. Python 3 is used only by the optional content extraction command below.

PostgreSQL is private; application ports bind only to loopback. Host nginx terminates HTTPS. The admin image's nginx serves static files and does not replace your host proxy. Runtime Compose has no build directives and disables pulling images.

## 1. Build and package locally

Run from the source workspace root. Prerequisites: Docker with Compose v2/BuildKit, Bash, Python 3.9+, tar, gzip, and sha256sum. An ARM build machine needs Docker's AMD64 emulation configured. The build requires package/image registries and Google Fonts access.

```bash
cp deploy/build.env.example deploy/build.env
mkdir -p deploy/secrets
touch deploy/secrets/sentry_auth_token
chmod 600 deploy/build.env deploy/secrets/sentry_auth_token
```

Edit `deploy/build.env` with the real website/admin HTTPS origins (no paths or trailing slashes), iiko image hostname, and optional public Sentry settings. These are compiled into images; runtime settings must match. Changes require a new release. Production passwords are not needed to build.

Optional source-map upload token: put it in `deploy/secrets/sentry_auth_token`, otherwise leave that file empty. It is mounted as a build secret and excluded from the deployment bundle.

```bash
./scripts/package-production.sh 2026-11-04.1
```

Choose a new release tag for each release; never reuse a deployed tag. The script builds all three applications for `linux/amd64`, packages PostgreSQL 17 under the same release tag, verifies image architectures, and writes `deploy/releases/<tag>/` containing:

- `images.tar.gz`: all four Docker images, ready for `docker load`.
- `deployment.tar.gz`: image-only Compose, nginx example, instructions, image IDs, release tag, and env templates populated with the public build settings.
- `SHA256SUMS`: checksums for both archives.

Existing release directories are never overwritten. Failed packaging removes temporary output; Docker build caches/images may remain. No credentials, local env files, build configuration, or source directories are included in the deployment bundle. Keep your local build env files out of version control. Avoid exporting conflicting build variables in the shell: Compose gives shell variables precedence over the env file.

Builds do not need the production backend. Next.js may log fallback content-fetch failures during prerendering; live API requests use `http://backend:3000/api/v1` at runtime. Existing application code controls content revalidation.

## 2. Transfer and load on the server

Create a private deployment directory once, then transfer the release (replace the host and tag):

```bash
ssh user@server 'mkdir -p /opt/citypark && chmod 700 /opt/citypark'
scp deploy/releases/2026-11-04.1/{images.tar.gz,deployment.tar.gz,SHA256SUMS} user@server:/opt/citypark/
```

Use a directory writable by your SSH user, or have the server administrator create `/opt/citypark` first. On the server:

```bash
cd /opt/citypark
sha256sum -c SHA256SUMS
# Continue only if BOTH checksums pass.
docker load --input images.tar.gz
tar -xzf deployment.tar.gz
```

The archive includes PostgreSQL, so startup needs no registry access. Runtime integrations such as iiko and Moneta still need network access.

## 3. Configure and start on the server

On first deployment only:

```bash
umask 077
cp deploy/production.env.example deploy/production.env
cp deploy/backend.env.example deploy/backend.env
cp deploy/website.env.example deploy/website.env
openssl rand -hex 32  # database password
openssl rand -hex 32  # JWT secret
```

Fill the required credentials in `deploy/production.env`. Its release tag and public settings already match the packaged images. Use a hex database password because it is inserted into a PostgreSQL URI without encoding. Use strong, unique admin credentials; `ADMIN_PHONE` is required. Admin creation happens once; editing these variables does not reset an existing account. Do not reuse development credentials.

`backend.env` configures iiko, Moneta, Telegram, hotel services, and optional customer authentication/email. Payment defaults use live Moneta; set merchant credentials and the provider callback to `https://<website>/api/v1/webhooks/moneta` (check URL: `/api/v1/webhooks/moneta/check`). Keep customer authentication disabled until its mail/OAuth settings are ready. `website.env` holds server-only integrations. Runtime settings require container recreation, not just restart.

Define this helper in each server shell, from `/opt/citypark`:

```bash
dc() { docker compose --env-file deploy/production.env -f compose.production.yml "$@"; }
dc config --quiet
dc up -d --no-build --pull never --wait
dc ps
dc logs --tail=100 backend website admin
```

The backend waits for PostgreSQL, runs embedded migrations, and ensures the initial admin exists. Website and admin wait for backend health. Health checks test process/HTTP availability, not every integration. Restart policies restart exited processes, not merely unhealthy containers. Configure external monitoring for the public website and API.

## Existing nginx

Adapt `deploy/nginx/citypark.conf.example` into your host nginx configuration. Replace both domains and certificate paths with your existing values. Reuse existing server blocks where appropriate; do not install duplicate blocks for the same hostname. The `map` belongs in the `http` context. This example assumes nginx is directly exposed to clients; adapt real-IP handling if a CDN sits in front.

| Public route | Upstream |
| --- | --- |
| Website `/` | `127.0.0.1:3001` |
| Website `/api/v1/` and `/uploads/` | `127.0.0.1:3002` |
| Admin subdomain `/` | `127.0.0.1:3003` |
| Admin subdomain `/api/v1/` | `127.0.0.1:3002` |

API paths are preserved. The admin uses an absolute URL on its own origin, including for live-order WebSockets. Uploaded media URLs use the website origin. Forwarded host and scheme preserve HTTPS Server Actions and customer redirects. Buffering and proxy caching are disabled for streaming and authenticated traffic. The 12 MB body limit accommodates the backend's 10 MB upload limit plus multipart overhead.

```bash
sudo nginx -t
sudo systemctl reload nginx
```

## Acceptance checks on the server

- Check `https://<website>/api/v1/health` and `https://<admin>/api/v1/health` return success.
- Open the website, menu, room pages, and images; inspect logs for failed API calls.
- Sign into the admin, reload a nested route, upload a photo, and verify the resulting website `/uploads/` URL.
- Verify the live orders WebSocket upgrades to HTTP 101, and reconnects after a backend restart.
- Verify customer redirects/cookies and payment callbacks with an authorized test transaction before accepting orders.
- Inspect `dc ps` and `ss -lnt`: ports 3001–3003 must bind to `127.0.0.1`; this stack must not publish 5432.
- Recreate containers with `dc up -d --no-build --pull never --force-recreate --wait` and confirm database records and uploads persist.

## Initial content

Migrations do not populate the production content bundle. Seed only when deliberately initializing or replacing content; it is not a startup step. Stop application writes and back up first. The bundled restore replaces selected content tables.

The website image contains the required public images. Extract them without starting a container (run this block in Bash); the source checkout is not needed. The temporary directory is readable by the backend's non-root user.

```bash
(
  set -euo pipefail
  seed_dir=$(mktemp -d)
  seed_container=
  trap 'if [ -n "$seed_container" ]; then docker rm "$seed_container" >/dev/null; fi; rm -rf -- "$seed_dir"' EXIT
  chmod 755 "$seed_dir"
  seed_image=$(dc config --format json | python3 -c 'import json,sys; print(json.load(sys.stdin)["services"]["website"]["image"])')
  seed_container=$(docker create --pull never "$seed_image")
  docker cp "$seed_container:/app/public/images/." "$seed_dir/"
  docker rm "$seed_container" >/dev/null
  seed_container=
  dc stop website admin backend
  dc run --rm --no-deps --entrypoint /app/content-seed \
    -v "$seed_dir:/site-images:ro" backend \
    check --bundle /app/seeds/production --images /site-images
  dc run --rm --no-deps --entrypoint /app/content-seed \
    -v "$seed_dir:/site-images:ro" backend \
    restore --replace --bundle /app/seeds/production --images /site-images
  dc up -d --no-build --pull never --wait
)
```

If seeding fails, applications remain stopped for inspection; PostgreSQL remains running.

## Backups and upgrades

Keep database and uploaded-file backups together, with matching application revisions and protected copies of configuration. Copy backups off the server. This example stops writers for a consistent maintenance-window backup:

```bash
mkdir -p deploy/backups
chmod 700 deploy/backups
dc stop website admin backend
dc exec -T postgres pg_dump -U citypark -d citypark -Fc > deploy/backups/database.dump
dc run --rm -T --no-deps --entrypoint tar backend -C /data/uploads -czf - . > deploy/backups/uploads.tar.gz
dc up -d --no-build --pull never --wait
```

Use a new dated backup directory for each real backup; these example filenames overwrite prior files. Protect permissions and verify both commands succeeded before relying on a backup. To restore during maintenance, with PostgreSQL running and application containers stopped:

```bash
dc exec -T postgres pg_restore -U citypark -d citypark --clean --if-exists --exit-on-error < deploy/backups/database.dump
dc run --rm -T --no-deps --user root --entrypoint sh backend -c \
  'find /data/uploads -mindepth 1 -delete && tar -xzf - -C /data/uploads && chown -R nobody /data/uploads' < deploy/backups/uploads.tar.gz
dc up -d --no-build --pull never --wait
```

The restore replaces database objects and uploaded files. Rehearse it on a separate server before relying on it. Never use `down -v` unless deliberately deleting all production data.

For upgrades, create a new uniquely tagged release on the build machine. On the server, back up first, verify checksums, load the new image archive, then extract its deployment bundle over the deployment directory. The bundle contains only templates and does not overwrite your `.env` files. Review configuration changes, update `RELEASE_TAG` in `deploy/production.env` to match `RELEASE`, and run `dc up -d --no-build --pull never --wait`. Check logs and public routes. Reuse the same deployment directory and Compose project name so existing volumes stay attached.

Keep the previous archives and tags. To roll back, load the previous archive if needed, restore its deployment bundle and matching configuration, set the previous release tag, and recreate containers. PostgreSQL 17 is also tagged per release so rollback selects the packaged image. Migrations run automatically: reverting images alone may not undo schema changes. If incompatible, restore the matching database/uploads backup with the previous application revision. PostgreSQL major-version upgrades need a separate migration procedure. Changing `POSTGRES_PASSWORD` in an env file does not change the password inside an initialized database.
