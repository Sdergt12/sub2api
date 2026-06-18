# Security Audit 2026-06-18

This audit records exposed risk areas found during the VPS and workspace review.
No secrets are printed here.

## High Risk

- Local workspace `info.txt` contains live GitHub and Cloudflare credentials.
  Move these values to a secret store or environment variables and rotate them.
- VPS Nginx config for `ai.gunddam.dpdns.org` contains an inline embed token.
  Move it to a generated snippet or environment-backed render step and rotate it.
- VPS backup files include `.env` and `.env.bak*` material under
  `/root/sub2api/**`. Treat those backups as sensitive and restrict retention.

## Medium Risk

- `.github/workflows/aux-images.yml` needs `packages: write` to publish GHCR
  images. Keep this workflow manual-only, and review package visibility and
  repository Actions permissions after the first successful run.
- GHCR tags are mutable by default. The new workflow prints image inspection
  data after push and labels each image with the source repository, commit SHA,
  version tag, and component name; keep those digests with the rollout record.
- `sub2api-risk-api` previously ran from `golang:1.22` with mounted source and
  `go run`. The new GHCR image flow removes source mounts from runtime.
- `sub2api-risk-web` and `sub2api-sign-web` previously ran `node:20`,
  `npm ci`, and Vite preview on the VPS. The new GHCR image flow moves builds
  to GitHub Actions and serves static assets from runtime images.
- Public services are exposed on `0.0.0.0` for `chatgpt2api`, `kiro-rs`, Nginx,
  SSH, and proxy components. Confirm each exposure is intentional.
- Docker bind mounts should stay minimal. The new risk compose keeps only the
  main application log path mounted read-only; sign should run without source or
  build-output mounts.
- Several auxiliary services still run as root or with elevated capabilities
  outside the Sub2API stack. Review them service by service before tightening.

## Low Risk

- Docker logs and rotated app logs are sizable. Keep current log rotation but
  add a retention review before disk pressure returns.
- Old preview files, screenshots, and one-off worker scripts in the local
  workspace are operationally confusing. Archive or delete once the online
  baseline is captured.

## Recommended Follow-up

- Rotate GitHub, Cloudflare, Nginx embed, and historical `.env` credentials
  after the new GHCR pipeline is verified.
- Keep one rollback package per online service: compose file, env key list,
  image tag/digest, and container inspect output.
- Prefer GitHub Actions for image builds and keep the VPS as a pull-and-run
  host only.
