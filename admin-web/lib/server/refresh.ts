import "server-only";
type Tokens = { access_token: string; refresh_token: string };
const pending = new Map<string, Promise<Tokens | null>>();
export function rotateOnce(
  base: string,
  token: string,
): Promise<Tokens | null> {
  const existing = pending.get(token);
  if (existing) return existing;
  const promise = (async () => {
    const r = await fetch(base + "/v1/admin/auth/token/refresh", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: token }),
      cache: "no-store",
      signal: AbortSignal.timeout(10000),
    });
    if (!r.ok) return null;
    return (await r.json()).data as Tokens;
  })();
  pending.set(token, promise);
  void promise
    .finally(() => setTimeout(() => pending.delete(token), 3000))
    .catch(() => {});
  return promise;
}
