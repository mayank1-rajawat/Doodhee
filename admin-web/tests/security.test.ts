import test from "node:test";
import assert from "node:assert/strict";
import { allowed } from "../lib/allowlist";
test("BFF permits only declared admin methods and paths", () => {
  assert.equal(allowed("GET", "users"), true);
  assert.equal(allowed("POST", "auth/mfa/verify"), true);
  assert.equal(
    allowed("POST", "users/11111111-1111-1111-1111-111111111111/disable"),
    true,
  );
  assert.equal(
    allowed("POST", "sellers/11111111-1111-1111-1111-111111111111/approve"),
    false,
  );
  for (const path of [
    "../orders",
    "auth/../users",
    "orders/arbitrary-id/correct-delivery-result",
    "anything",
    "users/1/disable",
  ])
    assert.equal(allowed("POST", path), false);
  assert.equal(allowed("DELETE", "users"), false);
  assert.equal(allowed("GET", "auth/mfa/verify"), false);
});
