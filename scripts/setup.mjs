import fs from "node:fs";
import crypto from "node:crypto";
if (!fs.existsSync(".env"))
  fs.writeFileSync(
    ".env",
    `APP_ENV=development\nTESTING_MODE=false\nDATABASE_URL=postgres://doodhee:doodhee-local-only@127.0.0.1:55432/doodhee\nAUTH_KEY=${crypto.randomBytes(48).toString("hex")}\nALLOWED_ORIGINS=http://127.0.0.1:3000,http://localhost:3000\n`,
    { mode: 0o600 },
  );
const current = fs.readFileSync(".env", "utf8");
if (!current.includes("PUBLIC_HTTP_ADDR="))
  fs.appendFileSync(
    ".env",
    "PUBLIC_HTTP_ADDR=127.0.0.1:18080\nADMIN_HTTP_ADDR=127.0.0.1:28081\n",
  );
if (!/^TESTING_MODE=/m.test(current))
  fs.appendFileSync(".env", "TESTING_MODE=false\n");
console.log(
  "Local configuration is ready. Existing environment files were preserved.",
);
