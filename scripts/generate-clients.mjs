import fs from "node:fs";
import { execFileSync } from "node:child_process";
const root = new URL("../", import.meta.url);
const read = (p) => JSON.parse(fs.readFileSync(new URL(p, root), "utf8"));
const write = (p, v) => {
  fs.mkdirSync(new URL("./", new URL(p, root)), { recursive: true });
  fs.writeFileSync(new URL(p, root), v);
};
for (const name of ["admin", "public"]) {
  const spec = read(`backend/api/openapi/${name}-v1.yaml`);
  const operations = Object.entries(spec.paths).flatMap(([path, verbs]) =>
    Object.entries(verbs).map(([method, op]) => ({
      id: op.operationId,
      path,
      method: method.toUpperCase(),
    })),
  );
  if (name === "admin") {
    write(
      "admin-web/generated/admin-api/operations.ts",
      "// Generated. Run make contracts.\nexport const operations = " +
        JSON.stringify(operations, null, 2) +
        ' as const;\nexport type OperationId=typeof operations[number]["id"];\n',
    );
    write(
      "admin-web/generated/admin-api/spec.json",
      JSON.stringify(spec, null, 2),
    );
    const ts = (s) =>
      s.$ref
        ? s.$ref.split("/").at(-1)
        : s.enum
          ? s.enum.map((v) => JSON.stringify(v)).join(" | ")
          : s.type === "boolean"
            ? "boolean"
            : s.type === "integer"
              ? "number"
              : s.type === "string"
                ? "string"
                : s.type === "array"
                  ? `Array<${ts(s.items)}>`
                  : "Record<string, unknown>";
    write(
      "admin-web/generated/admin-api/models.ts",
      "// Generated. Run make contracts.\n" +
        Object.entries(spec.components.schemas)
          .map(
            ([n, s]) =>
              `export interface ${n} {\n${Object.entries(s.properties ?? {})
                .map(
                  ([k, v]) =>
                    `  ${k}?: ${ts(v)}${v.nullable ? " | null" : ""};`,
                )
                .join("\n")}\n}`,
          )
          .join("\n"),
    );
  } else {
    write(
      "mobile-app/lib/generated/public_api/operations.dart",
      `// Generated. Run make contracts.\nconst publicOperations = <String, List<String>>{\n${operations.map((o) => `  '${o.id}': ['${o.method}', '${o.path}'],`).join("\n")}\n};\n`,
    );
    write(
      "mobile-app/lib/generated/public_api/models.dart",
      "// Generated. Run make contracts.\n" +
        Object.entries(spec.components.schemas)
          .map(
            ([name, s]) =>
              `class ${name}Dto {\n final Map<String,dynamic> json;\n const ${name}Dto(this.json);\n factory ${name}Dto.fromJson(Map<String,dynamic> json) => ${name}Dto(json);\n Map<String,dynamic> toJson() => json;\n${Object.entries(
                s.properties ?? {},
              )
                .map(([k, v]) => {
                  const t =
                    v.type === "boolean"
                      ? "bool"
                      : v.type === "integer"
                        ? "int"
                        : v.type === "string"
                          ? "String"
                          : "dynamic";
                  return ` ${t}${t === "dynamic" ? "" : "?"} get ${k.replace(/_([a-z])/g, (_, c) => c.toUpperCase())} => json['${k}']${t === "dynamic" ? "" : ` as ${t}?`};`;
                })
                .join("\n")}\n}`,
          )
          .join("\n"),
    );
  }
}
const dart = new URL(".tools/flutter/bin/dart", root).pathname;
execFileSync(
  fs.existsSync(dart) ? dart : "dart",
  ["format", new URL("mobile-app/lib/generated", root).pathname],
  { stdio: "pipe" },
);
