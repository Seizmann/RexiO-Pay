import { cookies } from "next/headers";
import { NextRequest } from "next/server";
import type { Merchant } from "@rexio-pay/shared-types";
import { BACKEND_KEY_COOKIE } from "@/lib/constants";

/**
 * Connect / disconnect a merchant API key.
 *
 * Gap flag #1: the backend has no merchant signup and no first-key bootstrap,
 * so the merchant pastes their rk_live_... key here once. The key is validated
 * against the backend (GET /v1/branding is the cheapest authenticated call)
 * and then stored in an httpOnly cookie that only the /api/backend proxy reads.
 */

const BACKEND_URL =
  process.env.API_BASE_URL ?? process.env.NEXT_PUBLIC_API_BASE_URL ?? "";

export async function POST(req: NextRequest) {
  if (!BACKEND_URL) {
    return Response.json(
      { error: { code: "internal_error", message: "Backend URL is not configured" } },
      { status: 500 },
    );
  }

  let key = "";
  try {
    const body = (await req.json()) as { api_key?: string };
    key = (body.api_key ?? "").trim();
  } catch {
    return Response.json(
      { error: { code: "invalid_request", message: "Request body must be JSON" } },
      { status: 400 },
    );
  }

  if (!key.startsWith("rk_") || key.length < 20) {
    return Response.json(
      { error: { code: "invalid_request", message: "That does not look like a RexiO Pay API key" } },
      { status: 400 },
    );
  }

  try {
    const res = await fetch(`${BACKEND_URL}/v1/branding`, {
      headers: { Authorization: `Bearer ${key}` },
      cache: "no-store",
    });
    if (!res.ok) {
      return Response.json(
        {
          error: {
            code: "unauthorized",
            message: "The RexiO Pay API rejected that key. Check it and try again.",
          },
        },
        { status: 401 },
      );
    }
    const merchant = (await res.json()) as Merchant;
    const cookieStore = await cookies();
    cookieStore.set(BACKEND_KEY_COOKIE, key, {
      httpOnly: true,
      sameSite: "lax",
      secure: process.env.NODE_ENV === "production",
      path: "/",
      maxAge: 60 * 60 * 24 * 30, // 30 days
    });
    return Response.json({
      merchant_id: merchant.id,
      merchant_name: merchant.name,
      plan_id: merchant.plan_id,
    });
  } catch {
    return Response.json(
      { error: { code: "internal_error", message: "Could not reach the RexiO Pay API" } },
      { status: 502 },
    );
  }
}

export async function DELETE() {
  const cookieStore = await cookies();
  cookieStore.delete(BACKEND_KEY_COOKIE);
  return Response.json({ ok: true });
}
