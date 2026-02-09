import { useSettingsStore, type Edition } from "@/store/settingsStore";

export function useEdition() {
  const edition = useSettingsStore((s) => s.edition);

  return {
    edition,
    isEnterprise: edition === "enterprise",
    isOSS: edition === "oss",
    isUnknown: edition === "unknown",
    requiresEnterprise: (feature: string) => {
      if (edition === "enterprise") return true;
      console.debug(`Feature "${feature}" requires NGINX Gateway Fabric Enterprise`);
      return false;
    },
  };
}

export type { Edition };
