# Auxiliary Images Deployment

This project builds auxiliary runtime images through GitHub Actions and GHCR.
The VPS should pull published images and should not be used as the normal build
host.

## Images

- `ghcr.io/sdergt12/sub2api-risk-api:<tag>`
- `ghcr.io/sdergt12/sub2api-risk-web:<tag>`
- `ghcr.io/sdergt12/sub2api-sign-api:<tag>`
- `ghcr.io/sdergt12/sub2api-sign-web:<tag>`

## Publish

1. Open the `Auxiliary Images` workflow in GitHub Actions.
2. Run it manually with:
   - `tag`: a deployment tag, for example `aux-20260618.1`
   - `component`: `all`, or one of `risk-api`, `risk-web`, `sign-api`, `sign-web`
3. Record the workflow run URL, image tag, and image digests.

## VPS rollout

Run these commands from the relevant service directory on the VPS after the
images have been published.

```bash
export AUX_IMAGE_TAG=aux-20260618.1

cd /root/sub2api/sub2api-risk
docker compose -f docker-compose.risk.yml pull
docker compose -f docker-compose.risk.yml up -d
curl -fsS http://127.0.0.1:8091/healthz
curl -fsS http://127.0.0.1:5173/external/risk/ >/dev/null

cd /root/sub2api/sub2api-sign
docker compose -f docker-compose.sign.yml pull
docker compose -f docker-compose.sign.yml up -d
curl -fsS http://127.0.0.1:8092/healthz
curl -fsS http://127.0.0.1:4174/external/sign/ >/dev/null
```

Before switching containers, save the current compose files, env files, Nginx
site files/snippets, and container inspect output in a timestamped rollback
directory.

## Rollback

1. Restore the previous compose files or set `AUX_IMAGE_TAG` back to the last
   known good tag.
2. Run `docker compose pull` and `docker compose up -d` for the affected
   service.
3. Verify `/healthz`, the public Nginx route, and recent container logs.

Do not remove the old images until the replacement has run cleanly through one
monitoring window.
