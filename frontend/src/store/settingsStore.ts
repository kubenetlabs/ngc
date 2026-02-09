import { create } from "zustand";
import { persist } from "zustand/middleware";

export type Edition = "oss" | "enterprise" | "unknown";
export type Theme = "light" | "dark";

interface SettingsState {
  theme: Theme;
  edition: Edition;
  defaultNamespace: string;
  setTheme: (theme: Theme) => void;
  setEdition: (edition: Edition) => void;
  setDefaultNamespace: (ns: string) => void;
  toggleTheme: () => void;
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set, get) => ({
      theme: "dark",
      edition: "unknown",
      defaultNamespace: "default",
      setTheme: (theme) => {
        document.documentElement.classList.toggle("dark", theme === "dark");
        set({ theme });
      },
      setEdition: (edition) => set({ edition }),
      setDefaultNamespace: (ns) => set({ defaultNamespace: ns }),
      toggleTheme: () => {
        const next = get().theme === "dark" ? "light" : "dark";
        document.documentElement.classList.toggle("dark", next === "dark");
        set({ theme: next });
      },
    }),
    { name: "ngf-console-settings" },
  ),
);
