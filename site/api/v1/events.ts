/**
 * Analytics ingest endpoint — Vercel Edge Function
 *
 * Accepts POST /v1/events with mine's anonymous usage payload,
 * validates it, and forwards to PostHog Cloud.
 *
 * Privacy: client IP is never forwarded ($ip: null sent to PostHog).
 * PostHog errors are swallowed — the endpoint always returns 2xx on
 * valid input to match the client's fail-silent behaviour.
 */

export const config = { runtime: "edge" };

interface MinePayload {
  install_id: string;
  version: string;
  os: string;
  arch: string;
  command: string;
  date: string;
}

const REQUIRED_FIELDS: (keyof MinePayload)[] = [
  "install_id",
  "version",
  "os",
  "arch",
  "command",
  "date",
];

const POSTHOG_CAPTURE_URL = "https://us.i.posthog.com/capture/";

export default async function handler(req: Request): Promise<Response> {
  // Only accept POST
  if (req.method !== "POST") {
    return new Response("Method Not Allowed", { status: 405 });
  }

  // Check POSTHOG_API_KEY is configured
  const apiKey = process.env.POSTHOG_API_KEY;
  if (!apiKey) {
    return new Response("Internal Server Error: analytics not configured", {
      status: 500,
    });
  }

  // Parse and validate the request body
  let payload: MinePayload;
  try {
    const body = await req.json();
    if (typeof body !== "object" || body === null || Array.isArray(body)) {
      return new Response("Bad Request: payload must be a JSON object", {
        status: 400,
      });
    }
    payload = body as MinePayload;
  } catch {
    return new Response("Bad Request: invalid JSON", { status: 400 });
  }

  // Validate required fields are present and non-empty strings
  for (const field of REQUIRED_FIELDS) {
    const value = payload[field];
    if (typeof value !== "string" || value.trim() === "") {
      return new Response(
        `Bad Request: missing or empty required field: ${field}`,
        { status: 400 }
      );
    }
  }

  // Forward to PostHog — errors are swallowed to match client fail-silent behaviour
  const posthogPayload = {
    api_key: apiKey,
    event: "command_run",
    distinct_id: payload.install_id,
    properties: {
      version: payload.version,
      os: payload.os,
      arch: payload.arch,
      command: payload.command,
      date: payload.date,
      $ip: null, // Suppress IP storage — privacy commitment
    },
  };

  try {
    await fetch(POSTHOG_CAPTURE_URL, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(posthogPayload),
    });
  } catch {
    // PostHog unreachable — still return 202 to caller
  }

  return new Response(null, { status: 202 });
}
