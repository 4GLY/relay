import { describe, expect, it } from "vitest";

import config from "./next.config.mjs";

describe("next config", () => {
  it("wires next-intl request config for standalone production builds", () => {
    expect(config).toMatchObject({
      reactStrictMode: true,
      output: "standalone",
    });

    expect(config).toEqual(
      expect.objectContaining({
        webpack: expect.any(Function),
      }),
    );
  });
});
