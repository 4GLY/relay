import { readFileSync } from "node:fs";
import { join } from "node:path";

import { describe, expect, it } from "vitest";

const filesToScan = [
  "app/projects/[projectId]/page.tsx",
  "app/projects/[projectId]/traces/page.tsx",
  "app/projects/[projectId]/graph/page.tsx",
  "app/projects/[projectId]/packet-builder/page.tsx",
  "app/style-memory/proposals.tsx",
  "components/relay-app-shell.tsx",
];

const forbiddenSnippets = [
  "Global navigation",
  "Project summary",
  "Decision Graph map",
  "Capture judgment traces to give Style Memory reviewable evidence.",
  "Couldn’t load approved heuristics.",
  "No rejected proposals yet.",
];

describe("i18n hardcoded copy guardrail", () => {
  it("keeps migrated product chrome out of TSX literals", () => {
    const offenders: string[] = [];

    for (const file of filesToScan) {
      const source = readFileSync(join(process.cwd(), file), "utf8");

      for (const snippet of forbiddenSnippets) {
        if (source.includes(snippet)) {
          offenders.push(`${file}: ${snippet}`);
        }
      }
    }

    expect(offenders).toEqual([]);
  });
});
