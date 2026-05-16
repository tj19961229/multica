import { describe, it, expect } from "vitest";
import { formatTokenCount } from "./tokens";

describe("formatTokenCount", () => {
  it("renders raw int below 1000", () => {
    expect(formatTokenCount(0)).toBe("0");
    expect(formatTokenCount(999)).toBe("999");
  });
  it("renders 1.X k for thousands", () => {
    expect(formatTokenCount(1_500)).toBe("1.5k");
    expect(formatTokenCount(12_500)).toBe("12.5k");
  });
  it("renders 1.X M for millions", () => {
    expect(formatTokenCount(1_500_000)).toBe("1.5M");
  });
});
