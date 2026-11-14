import AdminApp from "@/components/admin-app";
import { requireSessionCookie } from "@/lib/server/dal";
export default async function Page({
  params,
}: {
  params: Promise<{ path?: string[] }>;
}) {
  const { path = [] } = await params;
  if (!["login", "mfa"].includes(path[0])) await requireSessionCookie();
  return <AdminApp path={path} />;
}
