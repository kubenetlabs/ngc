import { NavLink } from "react-router-dom";
import {
  LayoutDashboard,
  Network,
  Route,
  Shield,
  KeyRound,
  BarChart3,
  Cpu,
  GitFork,
  Cloud,
  ArrowRightLeft,
  Wrench,
  ScrollText,
  Settings,
  Server,
} from "lucide-react";
import { useEdition } from "@/hooks/useEdition";
import type { LucideIcon } from "lucide-react";

interface NavItem {
  to: string;
  label: string;
  icon: LucideIcon;
  enterprise?: boolean;
}

const navItems: NavItem[] = [
  { to: "/", label: "Dashboard", icon: LayoutDashboard },
  { to: "/clusters", label: "Clusters", icon: Server },
  { to: "/gateways", label: "Gateways", icon: Network },
  { to: "/inference", label: "Inference", icon: Cpu },
  { to: "/routes", label: "Routes", icon: Route },
  { to: "/policies", label: "Policies", icon: Shield },
  { to: "/certificates", label: "Certificates", icon: KeyRound },
  { to: "/observability", label: "Observability", icon: BarChart3 },
  { to: "/diagnostics", label: "Diagnostics", icon: Wrench },
  { to: "/xc", label: "Distributed Cloud", icon: Cloud },
  { to: "/coexistence", label: "Coexistence", icon: GitFork },
  { to: "/migration", label: "Migration", icon: ArrowRightLeft },
  { to: "/audit", label: "Audit Log", icon: ScrollText },
  { to: "/settings", label: "Settings", icon: Settings },
];

export function Sidebar() {
  const { isEnterprise } = useEdition();

  return (
    <aside className="flex h-full w-64 flex-col border-r border-sidebar-border bg-sidebar text-sidebar-foreground">
      <div className="flex h-14 items-center gap-2 border-b border-sidebar-border px-4">
        <Network className="h-6 w-6 text-sidebar-primary" />
        <span className="text-lg font-semibold">NGF Console</span>
      </div>
      <nav className="flex-1 overflow-y-auto p-2">
        {navItems.map((item) => {
          const disabled = item.enterprise && !isEnterprise;
          return (
            <NavLink
              key={item.to}
              to={disabled ? "#" : item.to}
              onClick={disabled ? (e) => e.preventDefault() : undefined}
              className={({ isActive }) =>
                `flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors ${
                  disabled
                    ? "cursor-not-allowed text-muted-foreground opacity-50"
                    : isActive
                      ? "bg-sidebar-accent text-sidebar-accent-foreground"
                      : "text-sidebar-foreground hover:bg-sidebar-accent/50"
                }`
              }
            >
              <item.icon className="h-4 w-4" />
              {item.label}
              {item.enterprise && !isEnterprise && (
                <span className="ml-auto rounded bg-muted px-1.5 py-0.5 text-[10px] text-muted-foreground">
                  Enterprise
                </span>
              )}
            </NavLink>
          );
        })}
      </nav>
    </aside>
  );
}
