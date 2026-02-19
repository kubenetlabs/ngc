import { render, screen } from "@testing-library/react";
import { StatusBadge, StatusDot } from "../StatusBadge";
import type { Condition } from "@/types/gateway";

describe("StatusBadge", () => {
  const defaultCondition: Condition = {
    type: "Accepted",
    status: "True",
    reason: "Ready",
    message: "All good",
  };

  it("renders condition type and reason", () => {
    render(<StatusBadge condition={defaultCondition} />);
    expect(screen.getByText("Accepted: Ready")).toBeInTheDocument();
  });

  it("applies True status color class", () => {
    render(<StatusBadge condition={defaultCondition} />);
    const badge = screen.getByText("Accepted: Ready");
    expect(badge.className).toContain("emerald");
  });

  it("applies False status color class", () => {
    const condition: Condition = {
      ...defaultCondition,
      status: "False",
    };
    render(<StatusBadge condition={condition} />);
    const badge = screen.getByText("Accepted: Ready");
    expect(badge.className).toContain("red");
  });

  it("applies Unknown status color class", () => {
    const condition: Condition = {
      ...defaultCondition,
      status: "Unknown",
    };
    render(<StatusBadge condition={condition} />);
    const badge = screen.getByText("Accepted: Ready");
    expect(badge.className).toContain("zinc");
  });

  it("shows message as title attribute", () => {
    render(<StatusBadge condition={defaultCondition} />);
    expect(screen.getByTitle("All good")).toBeInTheDocument();
  });
});

describe("StatusDot", () => {
  it("renders label", () => {
    render(<StatusDot status="True" label="Healthy" />);
    expect(screen.getByText("Healthy")).toBeInTheDocument();
  });

  it("renders without label", () => {
    const { container } = render(<StatusDot status="False" />);
    // The outer span should only contain the dot span, no text content for a label
    const outerSpan = container.firstChild as HTMLElement;
    expect(outerSpan.children).toHaveLength(1);
    expect(screen.queryByText("Healthy")).not.toBeInTheDocument();
  });
});
