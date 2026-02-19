import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, it, expect, vi } from "vitest";
import { ErrorState } from "../ErrorState";

describe("ErrorState", () => {
  it("renders nothing when error is null", () => {
    const { container } = render(<ErrorState error={null} />);
    expect(container.firstChild).toBeNull();
  });

  it("renders error message from Error object", () => {
    render(<ErrorState error={new Error("Something went wrong")} />);
    expect(screen.getByText("Error: Something went wrong")).toBeInTheDocument();
  });

  it("renders custom message prop instead of error string", () => {
    render(
      <ErrorState
        error={new Error("raw error")}
        message="Failed to load gateways"
      />,
    );
    expect(screen.getByText("Failed to load gateways")).toBeInTheDocument();
    expect(screen.queryByText("raw error")).not.toBeInTheDocument();
  });

  it("renders retry button when onRetry provided", () => {
    render(
      <ErrorState error={new Error("fail")} onRetry={() => {}} />,
    );
    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();
  });

  it("hides retry button when onRetry not provided", () => {
    render(<ErrorState error={new Error("fail")} />);
    expect(screen.queryByRole("button", { name: "Retry" })).not.toBeInTheDocument();
  });

  it("calls onRetry when retry button is clicked", async () => {
    const onRetry = vi.fn();
    const user = userEvent.setup();

    render(<ErrorState error={new Error("fail")} onRetry={onRetry} />);
    await user.click(screen.getByRole("button", { name: "Retry" }));

    expect(onRetry).toHaveBeenCalledTimes(1);
  });
});
