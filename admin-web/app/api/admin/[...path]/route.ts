import { NextRequest, NextResponse } from "next/server";
import { allowed } from "@/lib/allowlist";
import { rotateOnce } from "@/lib/server/refresh";
const base = process.env.ADMIN_GATEWAY_URL ?? "http://127.0.0.1:28081";
const cookieOptions = {
  httpOnly: true,
  secure: process.env.NODE_ENV === "production",
  sameSite: "strict" as const,
  path: "/",
  maxAge: 8 * 3600,
};
type Payload = {
  data?: {
    access_token?: string;
    refresh_token?: string;
    mfa_challenge_id?: string;
    challenge_id?: string;
    expires_in?: number;
    testing_otp?: string;
    user?: unknown;
  };
  error?: { code: string; message: string };
};
async function handle(
  req: NextRequest,
  { params }: { params: Promise<{ path: string[] }> },
) {
  const path = (await params).path.join("/");
  if (!allowed(req.method, path))
    return NextResponse.json(
      { error: { code: "NOT_FOUND", message: "Route not found." } },
      { status: 404 },
    );
  let sameOrigin = false;
  try {
    const origin = new URL(req.headers.get("origin") ?? "");
    sameOrigin =
      origin.host === req.headers.get("host") &&
      (origin.protocol === "https:" || process.env.NODE_ENV !== "production");
  } catch {}
  if (req.method !== "GET" && !sameOrigin)
    return NextResponse.json(
      {
        error: {
          code: "CSRF_REJECTED",
          message: "Please reload and try again.",
        },
      },
      { status: 403 },
    );
  const auth = path.startsWith("auth/");
  let access = req.cookies.get("doodhee_access")?.value;
  let refresh = req.cookies.get("doodhee_refresh")?.value;
  if (!auth && !access && !refresh)
    return NextResponse.json(
      { error: { code: "UNAUTHENTICATED", message: "Please sign in." } },
      { status: 401 },
    );
  let body = req.method === "GET" ? undefined : await req.text();
  if (body && body.length > 65536)
    return new NextResponse(null, { status: 413 });
  if (path === "auth/token/refresh")
    body = JSON.stringify({ refresh_token: refresh });
  const call = () =>
    fetch(base + "/v1/admin/" + path + req.nextUrl.search, {
      method: req.method,
      headers: {
        "Content-Type": "application/json",
        ...(access ? { Authorization: "Bearer " + access } : {}),
        "X-Request-ID": crypto.randomUUID(),
        ...(req.headers.get("idempotency-key")
          ? { "Idempotency-Key": req.headers.get("idempotency-key")! }
          : {}),
      },
      body,
      cache: "no-store",
      signal: AbortSignal.timeout(15000),
    });
  try {
    let upstream = await call();
    let rotated = false;
    if (
      upstream.status === 401 &&
      (!auth || path === "auth/logout") &&
      refresh
    ) {
      const p = await rotateOnce(base, refresh);
      if (p) {
        access = p.access_token;
        refresh = p.refresh_token;
        rotated = true;
        upstream = await call();
      }
    }
    const text = await upstream.text();
    let payload: Payload = {};
    try {
      payload = JSON.parse(text);
    } catch {}
    const tokens = payload.data?.access_token;
    const nextRefresh = payload.data?.refresh_token;
    if (tokens) {
      access = tokens;
      refresh = nextRefresh;
      payload = { data: { user: payload.data?.user } };
      rotated = true;
    }
    const res =
      upstream.status === 204
        ? new NextResponse(null, { status: 204 })
        : NextResponse.json(payload, { status: upstream.status });
    const rid = upstream.headers.get("X-Request-ID");
    res.headers.set("Cache-Control", "no-store");
    if (rid) res.headers.set("X-Request-ID", rid);
    if (rotated && access && refresh) {
      res.cookies.set("doodhee_access", access, cookieOptions);
      res.cookies.set("doodhee_refresh", refresh, cookieOptions);
    }
    if (
      path === "auth/logout" ||
      upstream.status === 401 ||
      payload.error?.code === "ACCOUNT_DISABLED"
    ) {
      res.cookies.delete("doodhee_access");
      res.cookies.delete("doodhee_refresh");
    }
    return res;
  } catch {
    return NextResponse.json(
      {
        error: {
          code: "UPSTREAM_UNAVAILABLE",
          message: "The service is temporarily unavailable. Please retry.",
        },
      },
      { status: 503 },
    );
  }
}
export { handle as GET, handle as POST };
