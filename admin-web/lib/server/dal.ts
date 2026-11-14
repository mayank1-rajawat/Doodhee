import "server-only";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
// This guards the page shell. All private data access still goes through the BFF
// and the gateway's session, MFA, role and ownership checks.
export async function requireSessionCookie() {
  const jar = await cookies();
  if (!jar.has("doodhee_refresh")) redirect("/login");
}
