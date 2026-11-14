import fs from "node:fs";

// Check live routing/authentication without creating accounts or changing data.
// Successful authenticated flows are covered by TestAllAPIEndpoints on isolated data.
const root = new URL("../", import.meta.url);
const targets = [
  { name: "public-local", spec: "public", base: process.env.PUBLIC_API_URL ?? "http://127.0.0.1:18080" },
  { name: "admin-local", spec: "admin", base: process.env.ADMIN_API_URL ?? "http://127.0.0.1:28081" },
];
if (process.env.LAN_API_URL) targets.push({ name: "public-wifi", spec: "public", base: process.env.LAN_API_URL });
const values = {
  phone: "++++++++++", challenge_id: "00000000-0000-0000-0000-000000000000",
  code: "invalid", refresh_token: "api-smoke-invalid-token", product_id: "00000000-0000-0000-0000-000000000000",
  service_date: new Date(Date.now() + 86400000).toISOString().slice(0, 10),
  quantity: "1.000", delivery_address: "API smoke check", name: "Milk",
  price: "1.00", daily_capacity: "1.000", is_available: true,
  reason: "API smoke check", result: "delivered",
};
const jobs = [];
for (const target of targets) {
  const spec = JSON.parse(fs.readFileSync(new URL("backend/api/openapi/" + target.spec + "-v1.yaml", root), "utf8"));
  for (const path of ["/health/live", "/health/ready"]) jobs.push({ ...target, path, method: "GET", expected: 200, health: true });
  for (const [path, methods] of Object.entries(spec.paths)) {
    for (const [method, op] of Object.entries(methods)) {
      let expected = 401;
      if (op.operationId === "auth_request") expected = 422;
      if (op.operationId === "auth_mfa") expected = 404; // Unknown challenge: structured NOT_FOUND.
      const required = op.requestBody?.content?.["application/json"]?.schema?.required ?? [];
      const body = Object.fromEntries(required.map(k => [k, values[k]]));
      for (const k of required) if (values[k] === undefined) throw new Error("Missing smoke fixture field: " + k);
      jobs.push({ ...target, path, method: method.toUpperCase(), expected, body });
    }
  }
}
const results = await Promise.all(jobs.map(async job => {
  const started = performance.now();
  try {
    const response = await fetch(job.base.replace(/\/$/, "") + job.path.replaceAll("{id}", values.challenge_id), {
      method: job.method,
      headers: { "Content-Type": "application/json", "Idempotency-Key": "api-smoke-check-key" },
      body: job.method === "GET" ? undefined : JSON.stringify(job.body),
      signal: AbortSignal.timeout(5000),
    });
    const text = await response.text();
    let valid = response.status === job.expected;
    let errorCode;
    if (job.health) valid &&= text === "ok";
    else {
      const envelope = JSON.parse(text);
      errorCode = envelope.error?.code;
      valid &&= Boolean(errorCode && envelope.meta?.request_id && response.headers.get("x-request-id"));
    }
    return { target: job.name, method: job.method, path: job.path, expected: job.expected,
      actual: response.status, errorCode, passed: valid, durationMs: Math.round(performance.now() - started) };
  } catch (error) {
    return { target: job.name, method: job.method, path: job.path, expected: job.expected, passed: false, error: error.message };
  }
}));
const report = new URL("artifacts/api-verification/live-smoke.json", root);
fs.mkdirSync(new URL(".", report), { recursive: true });
fs.writeFileSync(report, JSON.stringify({ scope: "Live health, routing, malformed authentication input and missing-token rejection; no user mutations", results }, null, 2) + "\n");
const failed = results.filter(r => !r.passed);
console.log(JSON.stringify({ targets: targets.map(t => t.name), checks: results.length, failed: failed.length, report: report.pathname }));
if (failed.length) {
  console.error(JSON.stringify(failed, null, 2));
  process.exitCode = 1;
}
