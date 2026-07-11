# AVEX Platform — Replit Secrets Setup

This document explains how to securely configure secrets on Replit.

## Required Secrets

Open your Replit project, click the **lock icon** 🔒 in the left sidebar, and add the following keys:

| Secret Key | Value | How to Get |
|-----------|-------|------------|
| `JWT_SECRET` | Random 64-character hex string | Run `openssl rand -hex 32` in the Replit shell |
| `MAPBOX_ACCESS_TOKEN` | Your Mapbox public token | https://account.mapbox.com/access-tokens/ |

## Step-by-Step

### 1. Generate JWT_SECRET

In the Replit shell (bottom panel), run:

```bash
openssl rand -hex 32
```

Copy the output (a 64-character hex string like `a1b2c3...`).

### 2. Get MAPBOX_ACCESS_TOKEN

1. Go to https://account.mapbox.com/access-tokens/
2. Sign in or create a free account
3. Either use the **Default public token**, or click "Create a token" for a new one
4. Copy the token (starts with `pk.`)

### 3. Add Secrets to Replit

1. In the Replit sidebar, click the **lock icon** 🔒
2. For each secret, click "New secret":
   - Key: `JWT_SECRET`, Value: (paste the hex string from step 1)
   - Key: `MAPBOX_ACCESS_TOKEN`, Value: (paste the `pk....` token from step 2)
3. Click "Save"

### 4. Verify

Run the project. The `run.sh` script will fail with a clear error message if any secret is missing.

## Security Notes

- **Never commit secrets** to git. The `.env` file is gitignored.
- **Public Mapbox tokens are safe to expose** in browser code — they're designed for that. However, putting them in Replit Secrets keeps them out of git history.
- **JWT_SECRET must never be public**. Anyone with this secret can forge auth tokens.
- On Replit, secrets are encrypted at rest and only visible to people with project access.

## Optional Secrets

These have sensible defaults but can be overridden:

| Secret Key | Default | Purpose |
|-----------|---------|---------|
| `APP_LOG_LEVEL` | `info` | Log verbosity: `debug` \| `info` \| `warn` \| `error` |
| `OTEL_SAMPLER_RATIO` | `1.0` | Trace sampling ratio (0.0–1.0) |
| `BCRYPT_COST` | `12` | Password hashing cost factor (10–14) |

## Troubleshooting

### "Missing required environment variables: JWT_SECRET MAPBOX_ACCESS_TOKEN"
- Make sure you added both secrets via the Replit UI (lock icon).
- Restart the workflow after adding secrets.

### Map shows blank / "no token" error
- Verify `MAPBOX_ACCESS_TOKEN` starts with `pk.` (public token).
- Verify the token has the necessary scopes (default is fine).

### Login fails with "JWT verification error"
- The `JWT_SECRET` may have changed since tokens were issued. Log out and log back in.
