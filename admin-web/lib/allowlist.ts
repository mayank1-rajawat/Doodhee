import spec from "../generated/admin-api/spec.json";
export function allowed(method: string, path: string): boolean {
  return Object.entries(spec.paths).some(
    ([pattern, verbs]) =>
      new RegExp("^" + pattern.replace(/\{id\}/g, "[a-f0-9-]{36}") + "$").test(
        "/v1/admin/" + path,
      ) && method.toLowerCase() in verbs,
  );
}
