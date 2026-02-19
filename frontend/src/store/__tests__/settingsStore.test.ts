import { describe, it, expect, beforeEach } from "vitest";
import { useSettingsStore } from "../settingsStore";

describe("settingsStore", () => {
  beforeEach(() => {
    useSettingsStore.setState({
      theme: "dark",
      edition: "unknown",
      defaultNamespace: "default",
    });
  });

  it("initializes with dark theme", () => {
    expect(useSettingsStore.getState().theme).toBe("dark");
  });

  it("initializes with unknown edition", () => {
    expect(useSettingsStore.getState().edition).toBe("unknown");
  });

  it("initializes with default namespace", () => {
    expect(useSettingsStore.getState().defaultNamespace).toBe("default");
  });

  it("sets theme", () => {
    useSettingsStore.getState().setTheme("light");
    expect(useSettingsStore.getState().theme).toBe("light");
  });

  it("toggles theme from dark to light", () => {
    useSettingsStore.getState().toggleTheme();
    expect(useSettingsStore.getState().theme).toBe("light");
  });

  it("toggles theme from light to dark", () => {
    useSettingsStore.setState({ theme: "light" });
    useSettingsStore.getState().toggleTheme();
    expect(useSettingsStore.getState().theme).toBe("dark");
  });

  it("sets edition", () => {
    useSettingsStore.getState().setEdition("enterprise");
    expect(useSettingsStore.getState().edition).toBe("enterprise");
  });

  it("sets default namespace", () => {
    useSettingsStore.getState().setDefaultNamespace("kube-system");
    expect(useSettingsStore.getState().defaultNamespace).toBe("kube-system");
  });
});
