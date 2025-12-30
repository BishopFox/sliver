import { Docs } from "./docs";
import { Tutorials } from "./tutorials";

const buildAssetUrl = (path: string, version: string): string => {
  if (typeof window === "undefined") {
    return path;
  }
  const url = new URL(path, window.location.origin);
  if (version) {
    url.searchParams.set("v", version);
  }
  return url.toString();
};

export const fetchDocs = async (version: string): Promise<Docs> => {
  if (typeof window === "undefined") {
    return { docs: [] };
  }
  const res = await fetch(buildAssetUrl("/docs.json", version), {
    cache: "no-store",
  });
  return res.json();
};

export const fetchTutorials = async (version: string): Promise<Tutorials> => {
  if (typeof window === "undefined") {
    return { tutorials: [] };
  }
  const res = await fetch(buildAssetUrl("/tutorials.json", version), {
    cache: "no-store",
  });
  return res.json();
};
