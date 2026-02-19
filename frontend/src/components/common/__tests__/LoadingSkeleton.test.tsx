import { render } from "@testing-library/react";
import { describe, it, expect } from "vitest";
import {
  SkeletonCard,
  SkeletonTable,
  SkeletonText,
} from "../LoadingSkeleton";

describe("SkeletonCard", () => {
  it("renders without crashing", () => {
    const { container } = render(<SkeletonCard />);
    expect(container.firstChild).toBeTruthy();
  });

  it("has animate-pulse elements", () => {
    const { container } = render(<SkeletonCard />);
    const pulseElements = container.querySelectorAll(".animate-pulse");
    expect(pulseElements.length).toBeGreaterThan(0);
  });
});

describe("SkeletonTable", () => {
  it("renders with default 5 rows", () => {
    const { container } = render(<SkeletonTable />);
    const rows = container.querySelectorAll("tbody tr");
    expect(rows).toHaveLength(5);
  });

  it("renders with custom row count", () => {
    const { container } = render(<SkeletonTable rows={3} />);
    const rows = container.querySelectorAll("tbody tr");
    expect(rows).toHaveLength(3);
  });

  it("renders 4 columns per row", () => {
    const { container } = render(<SkeletonTable rows={1} />);
    const cells = container.querySelectorAll("tbody tr td");
    expect(cells).toHaveLength(4);
  });

  it("has animate-pulse elements in cells", () => {
    const { container } = render(<SkeletonTable rows={1} />);
    const pulseElements = container.querySelectorAll(
      "tbody td .animate-pulse",
    );
    expect(pulseElements).toHaveLength(4);
  });
});

describe("SkeletonText", () => {
  it("renders with default 3 lines", () => {
    const { container } = render(<SkeletonText />);
    const lines = container.querySelectorAll(".animate-pulse");
    expect(lines).toHaveLength(3);
  });

  it("renders with custom line count", () => {
    const { container } = render(<SkeletonText lines={5} />);
    const lines = container.querySelectorAll(".animate-pulse");
    expect(lines).toHaveLength(5);
  });

  it("renders with 1 line", () => {
    const { container } = render(<SkeletonText lines={1} />);
    const lines = container.querySelectorAll(".animate-pulse");
    expect(lines).toHaveLength(1);
  });
});
