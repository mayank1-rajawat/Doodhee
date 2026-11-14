"use client";
import { useCallback, useEffect, useState, useRef } from "react";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  LayoutDashboard,
  Users,
  Milk,
  ClipboardList,
  Activity,
  ShieldCheck,
  LogOut,
  Menu,
  RefreshCw,
  Search,
  ArrowUpRight,
  PackageCheck,
  Leaf,
} from "lucide-react";
type Row = Record<string, unknown>;
const sections = [
  ["dashboard", "Overview", LayoutDashboard],
  ["users", "Users", Users],
  ["products", "Products", Milk],
  ["orders", "Orders", ClipboardList],
  ["events", "Events", Activity],
  ["audit", "Audit Logs", ShieldCheck],
  ["bill-summary", "Bill Summary", PackageCheck],
] as const;
function text(v: unknown) {
  return v === null || v === undefined
    ? "—"
    : typeof v === "object"
      ? JSON.stringify(v)
      : String(v);
}
async function api(
  path: string,
  method = "GET",
  body?: Row,
  key: string = crypto.randomUUID(),
) {
  const r = await fetch("/api/admin/" + path, {
    method,
    headers: { "Content-Type": "application/json", "Idempotency-Key": key },
    body: body ? JSON.stringify(body) : undefined,
  });
  if (r.status === 204) return {};
  const p = await r.json();
  if (!r.ok)
    throw new Error(
      (p.error?.message ?? "Unable to complete this action.") +
        " " +
        (p.meta?.request_id ?? ""),
    );
  return p.data;
}
function Badge({ value }: { value: unknown }) {
  return (
    <span className={"badge " + text(value)}>
      {text(value).replaceAll("_", " ")}
    </span>
  );
}
function Brand({ onDark = false }: { onDark?: boolean }) {
  return (
    <span className={"brand " + (onDark ? "brand-on-dark" : "")}>
      <span className="brand-mark" aria-hidden="true">
        <Image
          src="/brand/doodhee-icon.png"
          alt=""
          width={48}
          height={48}
        />
      </span>
      <span className="brand-copy">
        Doodhee<small>ADMIN WORKSPACE</small>
      </span>
    </span>
  );
}
export default function AdminApp({ path }: { path: string[] }) {
  const requestKeys = useRef(new Map<string, string>());
  const route = path.join("/") || "dashboard";
  const router = useRouter();
  const [me, setMe] = useState<Row | null>(null);
  const [data, setData] = useState<unknown>(null);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);
  const [loading, setLoading] = useState(true);
  const [drawer, setDrawer] = useState(false);
  const [filters, setFilters] = useState<Row>({});
  const [modal, setModal] = useState<{
    title: string;
    endpoint: string;
    result?: boolean;
  } | null>(null);
  const login = route === "login" || route === "mfa";
  const endpoint = route === "audit" ? "audit-logs" : route;
  const load = useCallback(async () => {
    if (login) {
      setLoading(false);
      return;
    }
    setLoading(true);
    setError("");
    try {
      const u = await api("me");
      setMe(u);
      const q = new URLSearchParams(
        Object.fromEntries(
          Object.entries(filters)
            .filter(([, v]) => v !== "" && v !== undefined)
            .map(([k, v]) => [k, String(v)]),
        ),
      );
      setData(await api(endpoint + (q.size ? "?" + q : "")));
    } catch (e) {
      const msg = (e as Error).message;
      if (
        msg.includes("sign in") ||
        msg.includes("session") ||
        msg.includes("Sign in")
      )
        router.replace("/login");
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, [endpoint, filters, login, router]);
  useEffect(() => {
    const timer = setTimeout(() => {
      void load();
    }, 0);
    return () => clearTimeout(timer);
  }, [load]);
  const logout = async () => {
    setBusy(true);
    try {
      await api("auth/logout", "POST", {});
      router.replace("/login");
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  };
  const doAction = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    if (!modal) return;
    const f = new FormData(e.currentTarget);
    const body: Row = Object.fromEntries(f.entries());
    const fingerprint = modal.endpoint + JSON.stringify(body);
    const key = requestKeys.current.get(fingerprint) ?? crypto.randomUUID();
    requestKeys.current.set(fingerprint, key);
    setBusy(true);
    setError("");
    try {
      await api(modal.endpoint, "POST", body, key);
      requestKeys.current.delete(fingerprint);
      setNotice(
        modal.title + " completed. The action is recorded in the audit log.",
      );
      setModal(null);
      await load();
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  };
  if (login) return <Login onSuccess={() => router.replace("/dashboard")} />;
  const detail =
    path.length === 2 &&
    ["users", "orders", "products", "events"].includes(path[0]);
  const row = (detail ? data : null) as Row | null;
  const rows = (data as { items?: Row[] })?.items ?? [];
  const title = detail
    ? `${path[0].slice(0, -1)} details`
    : (sections.find((s) => s[0] === route)?.[1] ?? "Overview");
  const action = (label: string, url: string, result = false) => (
    <button
      key={url}
      onClick={() => setModal({ title: label, endpoint: url, result })}
    >
      {label}
    </button>
  );
  return (
    <div className="shell">
      <aside className={"sidebar " + (drawer ? "open" : "")}>
        <Link href="/dashboard" className="brand">
          <span className="brand-mark" aria-hidden="true">
            <Image
              src="/brand/doodhee-icon.png"
              alt=""
              width={48}
              height={48}
            />
          </span>
          <span className="brand-copy">
            Doodhee<small>ADMIN WORKSPACE</small>
          </span>
        </Link>
        <nav aria-label="Main navigation">
          {sections.map(([href, label, Icon]) => (
            <Link
              key={href}
              className={route === href ? "active" : ""}
              href={"/" + href}
              onClick={() => setDrawer(false)}
            >
              <Icon size={18} />
              {label}
            </Link>
          ))}
        </nav>
        <footer>
          Local milk. Clear records.
          <br />
          Doodhee Fresh design system
        </footer>
      </aside>
      <div className="workspace">
        <header className="topbar">
          <div className="identity">
            <button
              className="mobile-menu"
              aria-label="Open navigation"
              onClick={() => setDrawer(!drawer)}
            >
              <Menu size={20} />
            </button>
            <span className="muted">Operations workspace</span>
          </div>
          <div className="identity">
            <span className="badge">Admin</span>
            <span className="avatar">{text(me?.full_name).slice(0, 1)}</span>
            <span className="name">{text(me?.full_name)}</span>
            <button aria-label="Log out" disabled={busy} onClick={logout}>
              <LogOut size={17} />
            </button>
          </div>
        </header>
        <main>
          <div className="heading">
            <div>
              <div className="eyebrow">
                Doodhee / {detail ? path[0] : "Management"}
              </div>
              <h1 style={{ textTransform: "capitalize" }}>{title}</h1>
              <p className="muted">
                {route === "dashboard"
                  ? "A clear view of your local milk marketplace."
                  : "Review records and keep daily operations running smoothly."}
              </p>
            </div>
            <button
              onClick={() => void load()}
              disabled={loading}
              aria-label="Refresh data"
            >
              <RefreshCw size={16} />
              Refresh
            </button>
          </div>
          {error && (
            <div className="error" role="alert">
              {error}
            </div>
          )}
          {notice && (
            <div className="success" role="status">
              {notice}
            </div>
          )}
          {loading ? (
            <>
              <div className="skeleton" />
              <p className="muted" role="status">
                Loading your workspace…
              </p>
            </>
          ) : route === "dashboard" ? (
            <>
              <div className="stats">
                {[
                  ["users", "People", Users],
                  ["products", "Milk products", Milk],
                  ["orders", "Daily orders", ClipboardList],
                  ["outbox_events", "Business events", Activity],
                ].map(([key, label, Icon]) => (
                  <div className="card stat" key={String(key)}>
                    <Icon size={22} />
                    <strong>{text((data as Row)?.[String(key)])}</strong>
                    <span className="muted">{String(label)}</span>
                  </div>
                ))}
              </div>
              <div className="card">
                <h2>Keep the morning moving</h2>
                <p className="muted">
                  Milkman accounts can start immediately. Review daily deliveries
                  and manage account access here.
                </p>
                <div className="actions">
                  <Link href="/orders">
                    <button className="primary">
                      View daily deliveries
                      <ArrowUpRight size={17} />
                    </button>
                  </Link>
                  <Link href="/users">
                    <button>Manage users</button>
                  </Link>
                </div>
              </div>
            </>
          ) : detail && row ? (
            <>
              <div className="card">
                <h2>Record information</h2>
                <div className="detail-grid">
                  {Object.entries(row)
                    .filter(
                      ([k]) =>
                        ![
                          "secret_encrypted",
                          "refresh_token_hash",
                          "code_hash",
                          "approval_status",
                          "approval_reason",
                        ].includes(k),
                    )
                    .map(([k, v]) => (
                      <div key={k}>
                        <small>{k.replaceAll("_", " ")}</small>
                        {k === "status" ? (
                          <Badge value={v} />
                        ) : (
                          text(v)
                        )}
                      </div>
                    ))}
                </div>
                <div className="actions">
                  {path[0] === "users" &&
                    row.role !== "admin" &&
                    action(
                      row.is_enabled ? "Disable account" : "Enable account",
                      `users/${row.id}/${row.is_enabled ? "disable" : "enable"}`,
                    )}
                  {path[0] === "products" &&
                    action(
                      row.admin_suspended
                        ? "Activate product"
                        : "Suspend product",
                      `products/${row.id}/${row.admin_suspended ? "activate" : "suspend"}`,
                    )}
                  {path[0] === "orders" &&
                    action(
                      "Correct delivery result",
                      `orders/${row.id}/correct-delivery-result`,
                      true,
                    )}
                  {path[0] === "events" &&
                    row.status === "failed" &&
                    action("Retry failed event", `events/${row.id}/retry`)}
                </div>
              </div>
            </>
          ) : route === "bill-summary" ? (
            <>
              <Filters setFilters={setFilters} summary />
              <div className="stats">
                {["delivered_days", "delivered_litres", "total_amount"].map(
                  (k) => (
                    <div className="card stat" key={k}>
                      <span className="muted">{k.replaceAll("_", " ")}</span>
                      <strong>
                        {k === "total_amount" ? "₹ " : ""}
                        {text((data as Row)?.[k])}
                      </strong>
                    </div>
                  ),
                )}
              </div>
              <div className="card">
                <h2>By customer</h2>
                <pre>{JSON.stringify((data as Row)?.groups, null, 2)}</pre>
              </div>
            </>
          ) : (
            <>
              <Filters setFilters={setFilters} kind={route} />
              <div className="card" style={{ padding: 0 }}>
                {rows.length === 0 ? (
                  <div className="empty">
                    <Leaf size={36} />
                    <h2>No records to show</h2>
                    <p>New records will appear here as people use Doodhee.</p>
                    <button onClick={() => setFilters({})}>
                      Clear filters
                    </button>
                  </div>
                ) : (
                  <div className="table-wrap">
                    <table>
                      <thead>
                        <tr>
                          {Object.keys(rows[0])
                            .filter((k) =>
                              [
                                "id",
                                "full_name",
                                "role",
                                "phone",
                                "is_enabled",
                                "name",
                                "price",
                                "status",
                                "service_date",
                                "quantity",
                                "base_quantity",
                                "approved_extra_quantity",
                                "unit_price",
                                "price_source",
                                "total_amount",
                                "event_type",
                                "action",
                                "created_at",
                              ].includes(k),
                            )
                            .map((k) => (
                              <th key={k}>{k.replaceAll("_", " ")}</th>
                            ))}
                        </tr>
                      </thead>
                      <tbody>
                        {rows.map((r) => (
                          <tr key={text(r.id)}>
                            {Object.entries(r)
                              .filter(([k]) =>
                                [
                                  "id",
                                  "full_name",
                                  "role",
                                  "phone",
                                  "is_enabled",
                                  "name",
                                  "price",
                                  "status",
                                  "service_date",
                                  "quantity",
                                  "base_quantity",
                                  "approved_extra_quantity",
                                  "unit_price",
                                  "price_source",
                                  "total_amount",
                                  "event_type",
                                  "action",
                                  "created_at",
                                ].includes(k),
                              )
                              .map(([k, v]) => (
                                <td key={k}>
                                  {k === "id" && route !== "audit" ? (
                                    <Link
                                      href={
                                        "/" +
                                        route +
                                        "/" +
                                        v
                                      }
                                    >
                                      {text(v).slice(0, 8)}{" "}
                                      <ArrowUpRight size={12} />
                                    </Link>
                                  ) : k === "status" ? (
                                    <Badge value={v} />
                                  ) : k === "created_at" ? (
                                    new Date(text(v)).toLocaleDateString()
                                  ) : (
                                    text(v)
                                  )}
                                </td>
                              ))}
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                )}
              </div>
              {(data as { next_cursor?: string })?.next_cursor && (
                <button
                  className="next-page"
                  onClick={() =>
                    setFilters({
                      ...filters,
                      cursor: (data as { next_cursor: string }).next_cursor,
                    })
                  }
                >
                  Next page
                </button>
              )}
            </>
          )}
        </main>
      </div>
      {modal && (
        <div className="modal-backdrop">
          <section
            role="dialog"
            aria-modal="true"
            aria-labelledby="dialog-title"
            className="modal"
          >
            <h2 id="dialog-title">{modal.title}</h2>
            <p className="muted">
              This action updates the shared record and creates an audit entry.
            </p>
            <form onSubmit={doAction}>
              {modal.result && (
                <>
                  <p>Current status: {text(row?.status)}</p>
                  <label>
                    New delivery result
                    <select name="result">
                      <option value="delivered">Delivered</option>
                      <option value="not_delivered">Not delivered</option>
                    </select>
                  </label>
                  <label>
                    Non-delivery reason
                    <select name="reason_code">
                      <option value="other">Other</option>
                      <option value="customer_unavailable">
                        Buyer unavailable
                      </option>
                      <option value="seller_unavailable">
                        Seller unavailable
                      </option>
                      <option value="insufficient_milk">
                        Insufficient milk
                      </option>
                    </select>
                  </label>
                </>
              )}
              <label>
                Reason
                <textarea
                  autoFocus
                  name="reason"
                  minLength={3}
                  required
                  placeholder="Explain this action for the audit record"
                />
              </label>
              <div className="actions">
                <button type="button" onClick={() => setModal(null)}>
                  Cancel
                </button>
                <button className="primary" disabled={busy}>
                  {busy ? "Saving…" : "Confirm action"}
                </button>
              </div>
            </form>
          </section>
        </div>
      )}
    </div>
  );
}
function Filters({
  setFilters,
  summary = false,
  kind = "",
}: {
  setFilters: (r: Row) => void;
  summary?: boolean;
  kind?: string;
}) {
  return (
    <form
      className="card toolbar"
      onSubmit={(e) => {
        e.preventDefault();
        setFilters(Object.fromEntries(new FormData(e.currentTarget).entries()));
      }}
    >
      {kind === "users" && (
        <>
          <label>
            Phone
            <input name="phone" placeholder="Exact phone number" />
          </label>
          <label>
            Role
            <select name="role">
              <option value="">All roles</option>
              <option value="customer">Buyer</option>
              <option value="seller">Milkman</option>
              <option value="admin">Admin</option>
            </select>
          </label>
        </>
      )}
      {["orders", "products"].includes(kind) && (
        <label>
          Milkman ID
          <input name="seller_id" placeholder="Filter by Milkman" />
        </label>
      )}
      {kind === "orders" && (
        <label>
          Buyer ID
          <input name="customer_id" placeholder="Filter by buyer" />
        </label>
      )}
      {kind === "events" && (
        <label>
          Event type
          <input name="event_type" placeholder="e.g. OrderDelivered" />
        </label>
      )}
      {kind === "audit" && (
        <>
          <label>
            Admin ID
            <input name="admin_id" />
          </label>
          <label>
            Action
            <input name="action" />
          </label>
        </>
      )}
      {!summary && (
        <label>
          Status
          <input name="status" placeholder="e.g. pending" />
        </label>
      )}
      <label>
        From
        <input type="date" name="from" />
      </label>
      <label>
        To
        <input type="date" name="to" />
      </label>
      <button className="primary">
        <Search size={16} />
        Apply filters
      </button>
    </form>
  );
}
function Login({ onSuccess }: { onSuccess: () => void }) {
  const [stage, setStage] = useState("phone");
  const [challenge, setChallenge] = useState("");
  const [error, setError] = useState("");
  const [testingOtp, setTestingOtp] = useState("");
  const [busy, setBusy] = useState(false);
  const submit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setBusy(true);
    setError("");
    const f = new FormData(e.currentTarget);
    try {
      if (stage === "phone") {
        setTestingOtp("");
        const p = await api("auth/otp/request", "POST", {
          phone: f.get("phone"),
        });
        setChallenge(p.challenge_id);
        setTestingOtp(
          typeof p.testing_otp === "string" && /^[0-9]{6}$/.test(p.testing_otp)
            ? p.testing_otp
            : "",
        );
        setStage("otp");
      } else if (stage === "otp") {
        const p = await api("auth/otp/verify", "POST", {
          challenge_id: challenge,
          code: f.get("code"),
        });
        setChallenge(p.mfa_challenge_id);
        setTestingOtp("");
        setStage("mfa");
      } else {
        await api("auth/mfa/verify", "POST", {
          challenge_id: challenge,
          code: f.get("code"),
        });
        onSuccess();
      }
    } catch (err) {
      setError((err as Error).message);
    } finally {
      setBusy(false);
    }
  };
  return (
    <div className="login">
      <section className="login-story">
        <Brand onDark />
        <div>
          <div className="eyebrow" style={{ color: "#86efac" }}>
            Better mornings start here
          </div>
          <h1>
            Local milk.
            <br />
            Lasting trust.
          </h1>
          <p>
            One clear place to support your sellers, follow deliveries, and keep
            every daily record in sync.
          </p>
        </div>
        <footer>Fresh connections. Clear operations.</footer>
      </section>
      <section className="login-form">
        <form onSubmit={submit}>
          <span className="login-mark" aria-hidden="true">
            <Image
              src="/brand/doodhee-icon.png"
              alt=""
              width={58}
              height={58}
              priority
            />
          </span>
          <h1>
            {stage === "phone"
              ? "Welcome back"
              : stage === "otp"
                ? testingOtp ? "Verification code" : "Check your phone"
                : "One last security step"}
          </h1>
          <p className="muted">
            {stage === "phone"
              ? "Sign in with your registered admin phone number."
              : stage === "otp"
                ? testingOtp
                  ? "Use the testing code shown below."
                  : "Enter the six-digit code sent to your phone."
                : "Enter the current code from your authenticator app."}
          </p>
          {error && (
            <div className="error" role="alert">
              {error}
            </div>
          )}
          {stage === "otp" && testingOtp && (
            <div className="testing-otp" role="status">
              <span>Testing OTP</span>
              <strong>{testingOtp}</strong>
              <small>For testing only</small>
            </div>
          )}
          {stage === "phone" ? (
            <label>
              Phone number
              <input
                name="phone"
                type="tel"
                autoComplete="tel"
                placeholder="+91 …"
                required
              />
            </label>
          ) : (
            <label>
              {stage === "otp" ? "Verification code" : "Authenticator code"}
              <input
                key={stage}
                name="code"
                inputMode="numeric"
                pattern="[0-9]{6}"
                maxLength={6}
                autoComplete="one-time-code"
                required
                autoFocus
              />
            </label>
          )}
          <button className="primary" disabled={busy}>
            {busy
              ? "Please wait…"
              : stage === "phone"
                ? "Send verification code"
                : "Verify and continue"}
          </button>
          {stage !== "phone" && (
            <button
              type="button"
              style={{ marginTop: 12, width: "100%" }}
              disabled={busy}
              onClick={() => {
                setTestingOtp("");
                setChallenge("");
                setStage("phone");
              }}
            >
              Start again
            </button>
          )}
          <p className="muted" style={{ marginTop: 24, fontSize: 12 }}>
            Admin access requires phone verification and two-factor
            authentication.
          </p>
        </form>
      </section>
    </div>
  );
}
