import assert from "node:assert/strict";
import { readFileSync, readdirSync } from "node:fs";
import { join, relative } from "node:path";
import test from "node:test";
import { fileURLToPath } from "node:url";
import ts from "typescript";

test("visitor and staff screens use shared controls and field messages", () => {
  const apps = fileURLToPath(new URL("../apps/", import.meta.url));
  const violations: string[] = [];
  const sharedClasses: Record<string, string> = {
    button: "Button",
    "field-error": "FieldMessage",
    "form-help": "FieldMessage",
    "table-wrap": "TableRegion",
  };

  for (const app of ["admin", "visitor"]) {
    for (const entry of readdirSync(join(apps, app, "src"), {
      recursive: true,
      withFileTypes: true,
    })) {
      if (!entry.isFile() || !entry.name.endsWith(".tsx")) continue;
      const path = join(entry.parentPath, entry.name);
      const source = ts.createSourceFile(path, readFileSync(path, "utf8"), ts.ScriptTarget.Latest);

      function report(node: ts.Node, replacement: string) {
        const { line } = source.getLineAndCharacterOfPosition(node.getStart(source));
        violations.push(`${relative(apps, path)}:${line + 1} use ${replacement}`);
      }

      function inspect(node: ts.Node) {
        if (ts.isJsxOpeningElement(node) || ts.isJsxSelfClosingElement(node)) {
          const tag = node.tagName.getText(source);
          if (tag === "button") report(node, "Button");
          if (tag === "select") report(node, "Select");

          for (const attribute of node.attributes.properties) {
            if (
              !ts.isJsxAttribute(attribute) ||
              !attribute.initializer ||
              !ts.isStringLiteral(attribute.initializer)
            )
              continue;
            const name = attribute.name.getText(source);
            if (tag === "input" && name === "type" && attribute.initializer.text === "search")
              report(node, "SearchField");
            if (name === "className") {
              for (const className of attribute.initializer.text.split(/\s+/)) {
                const replacement = sharedClasses[className];
                if (replacement) report(node, replacement);
              }
            }
          }
        }
        ts.forEachChild(node, inspect);
      }

      inspect(source);
    }
  }
  assert.deepEqual(violations, []);
});
