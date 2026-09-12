import { cookies } from "next/headers";
import { NextRequest } from "next/server";
import { BACKEND_KEY_COOKIE } from "@/lib/constants";

/**
 * Same-origin proxy to the Go backend.
 *
 * The backend has no CORS middleware, so browsers cannot call it directly.
 * The merchant API key (rk_live_...) lives in an httpOnly cookie set by
 * /api/connect-key and never reaches browser JS. This route injects it into
 * the Authorization header server-side.
 */

const BACKEND_URL =
  process.env.API_BASE_URL ?? process.env.NEXT_PUBLIC_API_BASE_URL ?? "";

const FORWARDED_HEADERS = ["idempotency-key", "content-type"];

function buildHeaders(req: NextRequest, apiKey: string | undefined): Headers {
  const headers = new Headers();
  if (apiKey) headers.set("Authorization", `Bearer ${apiKey}`);
  for (const name of FORWARDED_HEADERS) {
    const value = req.headers.get(name);
    if (value) headers.set(name, value);
  }
  return headers;
}

async function proxy(req: NextRequest, path: string[]): Promise<Response> {
  if (!BACKEND_URL) {
    return Response.json(
      { error: { code: "internal_error", message: "Backend URL is not configured" } },
      { status: 500 },
    );
  }
  const isPublic =
    path[0] === "checkout" || (path[0] === "device" && path[1] === "pair");

  const cookieStore = await cookies();
  const apiKey = cookieStore.get(BACKEND_KEY_COOKIE)?.value;
  if (!apiKey && !isPublic) {
    return Response.json(
      { error: { code: "unauthorized", message: "Not connected to the RexiO Pay API" } },
      { status: 401 },
    );
  }

  const target = `${BACKEND_URL}/v1/${path.join("/")}${
    req.nextUrl.search // preserve query strings (pagination, filters)
  }`;

  try {
    const res = await fetch(target, {
      method: req.method,
      headers: buildHeaders(req, apiKey),
      body:
        req.method === "GET" || req.method === "HEAD"
          ? undefined
          : await req.text(),
      cache: "no-store",
    });

    const body = await res.text();
    return new Response(body, {
      status: res.status,
      headers: { "content-type": res.headers.get("content-type") ?? "application/json" },
    });
  } catch {
    return Response.json(
      { error: { code: "internal_error", message: "Could not reach the RexiO Pay API" } },
      { status: 502 },
    );
  }
}

type Ctx = { params: Promise<{ path: string[] }> };

export async function GET(req: NextRequest, ctx: Ctx) {
  const { path } = await ctx.params;
  return proxy(req, path);
}
export async function POST(req: NextRequest, ctx: Ctx) {
  const { path } = await ctx.params;
  return proxy(req, path);
}
export async function PATCH(req: NextRequest, ctx: Ctx) {
  const { path } = await ctx.params;
  return proxy(req, path);
}
export async function PUT(req: NextRequest, ctx: Ctx) {
  const { path } = await ctx.params;
  return proxy(req, path);
}
export async function DELETE(req: NextRequest, ctx: Ctx) {
  const { path } = await ctx.params;
  return proxy(req, path);
}
