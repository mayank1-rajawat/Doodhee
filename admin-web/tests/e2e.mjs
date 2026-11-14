import { chromium } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
import fs from "node:fs";
import crypto from "node:crypto";
import { execFileSync } from "node:child_process";
const root = new URL("../../", import.meta.url).pathname;
const env = Object.fromEntries(
  fs
    .readFileSync(root + ".env", "utf8")
    .trim()
    .split("\n")
    .filter((l) => l.includes("="))
    .map((l) => {
      const i = l.indexOf("=");
      return [l.slice(0, i), l.slice(i + 1)];
    }),
);
const phone = "+199" + Date.now().toString().slice(-9);
const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567";
const secret = Array.from(crypto.randomBytes(32), (v) => alphabet[v % 32]).join(
  "",
);
execFileSync(root + ".tools/go/bin/go", ["run", "./cmd/seed-admin"], {
  cwd: root + "backend",
  env: {
    ...process.env,
    ...env,
    ADMIN_PHONE: phone,
    ADMIN_TOTP_SECRET: secret,
  },
  stdio: "pipe",
});
function totp(secret) {
  let bits = "";
  for (const c of secret)
    bits += alphabet.indexOf(c).toString(2).padStart(5, "0");
  const bytes = [];
  for (let i = 0; i + 8 <= bits.length; i += 8)
    bytes.push(parseInt(bits.slice(i, i + 8), 2));
  const t = Buffer.alloc(8);
  t.writeBigUInt64BE(BigInt(Math.floor(Date.now() / 30000)));
  const h = crypto.createHmac("sha1", Buffer.from(bytes)).update(t).digest();
  const o = h[19] & 15;
  return ((h.readUInt32BE(o) & 0x7fffffff) % 1000000)
    .toString()
    .padStart(6, "0");
}
function code(file) {
  const lines = fs.readFileSync(file, "utf8").trim().split("\n");
  for (const l of lines.reverse()) {
    try {
      const r = JSON.parse(l);
      if (r.challenge_code) return r.challenge_code;
    } catch {}
  }
  throw new Error("Development OTP log not found");
}
const browser = await chromium.launch({
  executablePath: "/usr/bin/google-chrome",
  headless: true,
  args: ["--no-sandbox"],
});
const context = await browser.newContext({
  viewport: { width: 1440, height: 1000 },
});
const page = await context.newPage();
try {
  await page.goto("http://127.0.0.1:3000/login");
  await page.getByLabel("Phone number").fill(phone);
  await page.getByRole("button", { name: "Send verification code" }).click();
  await page.getByLabel("Verification code", { exact: true }).waitFor();
  await page
    .getByLabel("Verification code", { exact: true })
    .fill(code("/tmp/doodhee-admin-dev.log"));
  await page.getByRole("button", { name: "Verify and continue" }).click();
  await page.getByLabel("Authenticator code").waitFor();
  await page.getByLabel("Authenticator code").fill(totp(secret));
  await page.getByRole("button", { name: "Verify and continue" }).click();
  await page.getByRole("heading", { name: "Overview" }).waitFor();
  await page.waitForTimeout(600);
  const violations = (await new AxeBuilder({ page }).analyze()).violations;
  if (violations.length)
    throw new Error(
      "Accessibility violations: " + violations.map((v) => v.id).join(","),
    );
  fs.mkdirSync(root + "artifacts", { recursive: true });
  await page.screenshot({
    path: root + "artifacts/admin-dashboard.png",
    fullPage: true,
  });
  const base = "http://127.0.0.1:18080";
  async function call(path, method = "GET", body, token, key) {
    const r = await fetch(base + "/v1/" + path, {
      method,
      headers: {
        "Content-Type": "application/json",
        ...(token ? { Authorization: "Bearer " + token } : {}),
        ...(key ? { "Idempotency-Key": key } : {}),
      },
      body: body ? JSON.stringify(body) : undefined,
    });
    const p = await r.json();
    if (!r.ok) throw new Error(path + ": " + JSON.stringify(p));
    return p.data;
  }
  async function register(role, suffix) {
    const p = await call("auth/otp/request", "POST", {
      phone: '+188'+crypto.randomInt(100000000,999999999).toString(),
      role,
    });
    return call("auth/otp/verify", "POST", {
      challenge_id: p.challenge_id,
      code: code("/tmp/doodhee-public-dev.log"),
      full_name: "E2E " + role,
    });
  }
  const seller = await register("seller", "11"),
    buyer = await register("customer", "22");
  const product = await call(
    "seller/products",
    "POST",
    { name: "Fresh Cow Milk", price: "65.00", daily_capacity: "100.000" },
    seller.access_token,
  );
  const order = await call(
    "orders",
    "POST",
    {
      product_id: product.id,
      service_date: new Date(Date.now() + 86400000).toISOString().slice(0, 10),
      quantity: "1.500",
      delivery_address: "E2E house, test street",
    },
    buyer.access_token,
    crypto.randomUUID(),
  );
  await call(
    "seller/orders/" + order.id + "/accept",
    "POST",
    {},
    seller.access_token,
  );
  await call(
    "seller/orders/" + order.id + "/out-for-delivery",
    "POST",
    {},
    seller.access_token,
  );
  await call(
    "seller/orders/" + order.id + "/delivery-result",
    "POST",
    { result: "delivered" },
    seller.access_token,
    crypto.randomUUID(),
  );
  const summary = await call("bill-summary", "GET", null, buyer.access_token);
  if (summary.total_amount !== "97.50") throw new Error("Buyer total mismatch");
  await page.goto("http://127.0.0.1:3000/orders/" + order.id);
  await page.getByRole("heading", { name: "Record information" }).waitFor();
  await page.getByRole("button", { name: "Correct delivery result" }).click();
  await page.getByLabel("New delivery result").selectOption("not_delivered");
  await page
    .getByLabel("Reason", { exact: true })
    .fill("Delivery correction verified by admin");
  await page.getByRole("button", { name: "Confirm action" }).click();
  await page.locator('.success[role="status"]').waitFor();
  const corrected = await call(
    "orders/" + order.id,
    "GET",
    null,
    buyer.access_token,
  );
  if (corrected.status !== "not_delivered")
    throw new Error("Correction did not propagate");
  const finalSummary = await call(
    "bill-summary",
    "GET",
    null,
    buyer.access_token,
  );
  if (finalSummary.total_amount !== "0.00")
    throw new Error("Corrected total mismatch");
  const disable = await page.evaluate(async (id) => {
    const r = await fetch("/api/admin/users/" + id + "/disable", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ reason: "End-to-end disablement test" }),
    });
    return r.status;
  }, buyer.user.id);
  if (disable !== 200) throw new Error("Disable failed");
  const rejected = await fetch(base + "/v1/me", {
    headers: { Authorization: "Bearer " + buyer.access_token },
  });
  if (rejected.status !== 401 && rejected.status !== 403)
    throw new Error("Disabled account still authenticated");
  await page.goto("http://127.0.0.1:3000/users");
  await page.getByRole("heading", { name: "Users" }).waitFor();
  await page.screenshot({
    path: root + "artifacts/admin-users.png",
    fullPage: true,
  });
  console.log(
    "PASS: browser OTP + MFA, immediate Milkman access, public buyer/seller order flow, admin correction, matched totals, disablement.",
  );
} finally {
  await browser.close();
}
