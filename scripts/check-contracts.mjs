import fs from "node:fs";
import crypto from "node:crypto";
import { execFileSync } from "node:child_process";
const paths = [
  "backend/api/openapi/public-v1.yaml",
  "backend/api/openapi/admin-v1.yaml",
  "admin-web/generated/admin-api/spec.json",
  "admin-web/generated/admin-api/models.ts",
  "admin-web/generated/admin-api/operations.ts",
  "mobile-app/lib/generated/public_api/operations.dart",
  "mobile-app/lib/generated/public_api/models.dart",
];
const hash = (p) =>
  crypto.createHash("sha256").update(fs.readFileSync(p)).digest("hex");
const before = paths.map(hash);
execFileSync("make", ["contracts"], { stdio: "pipe" });
const after = paths.map(hash);
if (before.some((v, i) => v !== after[i])) {
  console.error(
    "Generated contracts were stale. Run make contracts and commit the updated generated files.",
  );
  process.exit(1);
}
console.log("Generated contracts are current.");
