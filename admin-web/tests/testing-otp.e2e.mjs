import { chromium, expect } from "@playwright/test";
import fs from "node:fs";

// Browser-only response fixtures; backend exposure/verification is tested against PostgreSQL.
const base = process.env.ADMIN_WEB_URL ?? "http://127.0.0.1:3000";
const browser = await chromium.launch({
  executablePath: "/usr/bin/google-chrome",
  headless: true,
  args: ["--no-sandbox"],
});
try {
  for (const enabled of [false, true]) {
    const context = await browser.newContext();
    const page = await context.newPage();
    let requests = 0;
    await page.route("**/api/admin/auth/**", async route => {
      const path = new URL(route.request().url()).pathname;
      if (path.endsWith("/otp/request")) {
        requests++;
        await route.fulfill({
          json: { data: { challenge_id: "test-challenge", expires_in: 300,
            ...(enabled && requests === 1 ? { testing_otp: "012345" } : {}) } },
        });
      } else if (path.endsWith("/otp/verify")) {
        await route.fulfill({ json: { data: { mfa_challenge_id: "test-mfa" } } });
      } else {
        throw new Error("Unexpected test request: " + path);
      }
    });
    await page.goto(base + "/login");
    await page.getByLabel("Phone number").fill("+15555550123");
    await page.getByRole("button", { name: "Send verification code" }).click();
    await expect(page.getByLabel("Verification code", { exact: true })).toBeVisible();
    if (enabled) {
      await expect(page.getByRole("status")).toContainText("Testing OTP");
      await expect(page.getByRole("status")).toContainText("012345");
      await expect(page.getByLabel("Verification code", { exact: true })).toHaveValue("");
      fs.mkdirSync(new URL("../../artifacts/", import.meta.url), { recursive: true });
      await page.screenshot({ path: new URL("../../artifacts/testing-otp-admin.png", import.meta.url).pathname });
    } else {
      await expect(page.locator(".testing-otp")).toHaveCount(0);
    }
    await page.getByLabel("Verification code", { exact: true }).fill("012345");
    await page.getByRole("button", { name: "Verify and continue" }).click();
    await expect(page.getByLabel("Authenticator code")).toBeVisible();
    await expect(page.locator(".testing-otp")).toHaveCount(0);
    await page.getByRole("button", { name: "Start again" }).click();
    await page.getByLabel("Phone number").fill("+15555550123");
    await page.getByRole("button", { name: "Send verification code" }).click();
    await expect(page.getByLabel("Verification code", { exact: true })).toBeVisible();
    await expect(page.locator(".testing-otp")).toHaveCount(0);
    await context.close();
  }
  console.log("PASS: admin testing OTP shown/hidden, leading zero retained, manual verification, MFA step and reset.");
} finally {
  await browser.close();
}
