import { Moon, Sun } from "lucide-react";
import { useSettingsStore } from "@/store/settingsStore";

export function Header() {
  const { theme, toggleTheme, edition } = useSettingsStore();

  return (
    <header className="flex h-14 items-center justify-between border-b border-border bg-background px-6">
      <div className="flex items-center gap-4">
        <h2 className="text-sm font-medium text-muted-foreground">
          NGINX Gateway Fabric
        </h2>
        <span
          className={`rounded-full px-2 py-0.5 text-xs font-medium ${
            edition === "enterprise"
              ? "bg-enterprise/20 text-enterprise"
              : edition === "oss"
                ? "bg-primary/20 text-primary"
                : "bg-muted text-muted-foreground"
          }`}
        >
          {edition === "enterprise" ? "Enterprise" : edition === "oss" ? "OSS" : "Detecting..."}
        </span>
      </div>
      <div className="flex items-center gap-2">
        <button
          onClick={toggleTheme}
          className="rounded-md p-2 text-muted-foreground hover:bg-accent hover:text-accent-foreground"
        >
          {theme === "dark" ? <Sun className="h-4 w-4" /> : <Moon className="h-4 w-4" />}
        </button>
      </div>
    </header>
  );
}
