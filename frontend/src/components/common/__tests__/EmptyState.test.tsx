import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { EmptyState } from "../EmptyState";

describe("EmptyState", () => {
  it("renders title and description", () => {
    render(
      <MemoryRouter>
        <EmptyState title="No data" description="Nothing here" />
      </MemoryRouter>,
    );
    expect(screen.getByText("No data")).toBeInTheDocument();
    expect(screen.getByText("Nothing here")).toBeInTheDocument();
  });

  it("renders action link when provided", () => {
    render(
      <MemoryRouter>
        <EmptyState
          title="No data"
          description="Nothing here"
          action={{ label: "Create", href: "/create" }}
        />
      </MemoryRouter>,
    );
    const link = screen.getByRole("link", { name: "Create" });
    expect(link).toBeInTheDocument();
    expect(link).toHaveAttribute("href", "/create");
  });

  it("does not render action when not provided", () => {
    render(
      <MemoryRouter>
        <EmptyState title="No data" description="Nothing here" />
      </MemoryRouter>,
    );
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });

  it("renders icon when provided", () => {
    function TestIcon(props: React.SVGProps<SVGSVGElement>) {
      return <svg data-testid="icon" {...props} />;
    }

    render(
      <MemoryRouter>
        <EmptyState title="No data" description="Nothing here" icon={TestIcon} />
      </MemoryRouter>,
    );
    expect(screen.getByTestId("icon")).toBeInTheDocument();
  });
});
