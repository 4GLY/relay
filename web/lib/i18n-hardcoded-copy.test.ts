import { readFileSync } from "node:fs";
import { join } from "node:path";

import { describe, expect, it } from "vitest";

const filesToScan = [
  "app/projects/[projectId]/page.tsx",
  "app/projects/[projectId]/traces/page.tsx",
  "app/projects/[projectId]/graph/page.tsx",
  "app/projects/[projectId]/packet-builder/page.tsx",
  "app/style-memory/page.tsx",
  "app/style-memory/proposals.tsx",
  "app/onboarding/onboarding-client.tsx",
  "app/settings/providers/page.tsx",
  "app/settings/api-keys/page.tsx",
  "components/relay-app-shell.tsx",
];

const forbiddenSnippets = [
  "Global navigation",
  "Project summary",
  "Decision Graph map",
  "Capture judgment traces to give Style Memory reviewable evidence.",
  "Couldn’t load approved heuristics.",
  "Couldn’t load proposals",
  "No rejected proposals yet.",
  "Settings navigation",
  "Pick a project",
  "Style Memory needs a project",
  "Something is wrong with the queue right now.",
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
