# Analytics Ingest API

This directory contains Vercel Edge Functions for the `mine` analytics backend.

## Endpoints

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/v1/events` | Receive anonymous usage pings from `mine` binaries |

## Architecture

```
mine binary → POST analytics.mine.rwolfe.io/v1/events
                  → Edge Function (validate, strip IP, transform)
                      → POST PostHog Capture API
```

The edge function:
- Validates the incoming payload (required fields, correct types)
- Strips the client IP — `$ip: null` is sent to PostHog, fulfilling the privacy commitment in the docs
- Forwards to PostHog Cloud under the `command_run` event name
- Always returns `202 Accepted` to the caller for valid requests; PostHog errors are swallowed silently to match the client's fail-silent behaviour

## Human Setup Steps

These steps must be completed manually before the endpoint is live.

### 1. Create a PostHog Cloud project

1. Go to [posthog.com](https://posthog.com) and sign up or log in
2. Create a new project (e.g. `mine`)
3. Copy the **Project API Key** — it starts with `phc_`

### 2. Add the environment variable in Vercel

1. Open the [Vercel dashboard](https://vercel.com) → select the `mine` project
2. Go to **Settings → Environment Variables**
3. Add a new variable:
   - **Name**: `POSTHOG_API_KEY`
   - **Value**: your `phc_...` key
   - **Environments**: Production (and Preview if desired)
4. Save — no redeployment needed for edge functions; the var is read at runtime

### 3. Add the analytics domain alias

1. In the Vercel dashboard → project → **Settings → Domains**
2. Add `analytics.mine.rwolfe.io` as a domain alias

### 4. Configure DNS

Add a CNAME record in your DNS provider:

| Type | Name | Value |
|------|------|-------|
| `CNAME` | `analytics` | `cname.vercel-dns.com` |

Once DNS propagates, `https://analytics.mine.rwolfe.io/v1/events` will route to the edge function.

## Payload Schema

The `mine` binary sends this JSON body on each command invocation (at most once per command per day):

```json
{
  "install_id": "uuid-v4",
  "version": "0.1.0",
  "os": "linux",
  "arch": "amd64",
  "command": "todo",
  "date": "2026-02-17"
}
```

All fields are required. The endpoint returns `400` if any are missing or empty.

## Response Codes

| Code | Meaning |
|------|---------|
| `202` | Event accepted and forwarded to PostHog |
| `400` | Malformed or missing required fields |
| `405` | Non-POST method |
| `500` | `POSTHOG_API_KEY` not configured |
