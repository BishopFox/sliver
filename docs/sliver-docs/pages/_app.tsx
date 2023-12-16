import Navbar from "@/components/navbar";
import "@/styles/globals.css";
import { Docs } from "@/util/docs";
import { SearchContext, SearchCtx } from "@/util/search-context";
import { Themes } from "@/util/themes";
import { NextUIProvider } from "@nextui-org/react";
import {
  HydrationBoundary,
  QueryClient,
  QueryClientProvider,
} from "@tanstack/react-query";
import { ThemeProvider as NextThemesProvider, useTheme } from "next-themes";
import type { AppProps } from "next/app";
import React from "react";

export default function App({ Component, pageProps }: AppProps) {
  // Initialize theme
  const { theme, setTheme } = useTheme();
  function getThemeState(): Themes {
    if (typeof window !== "undefined") {
      const loadedTheme = localStorage.getItem("theme");
      const currentTheme = loadedTheme ? (loadedTheme as Themes) : Themes.DARK;
      setTheme(currentTheme);
    }
    return Themes.DARK;
  }

  // Initialize search
  const [search, setSearch] = React.useState(new SearchCtx());

  // Initialize query client
  const [queryClient] = React.useState(() => new QueryClient());
  queryClient.prefetchQuery({
    queryKey: ["docs"],
    queryFn: async (): Promise<Docs> => {
      const res = await fetch("/docs.json");
      const docs: Docs = await res.json();
      search.addDocs(docs);
      return docs;
    },
  });

  return (
    <NextUIProvider>
      <NextThemesProvider attribute="class" defaultTheme={getThemeState()}>
        <QueryClientProvider client={queryClient}>
          <HydrationBoundary state={pageProps.dehydratedState}>
            <SearchContext.Provider value={search}>
              <Navbar />
              <Component {...pageProps} />
            </SearchContext.Provider>
          </HydrationBoundary>
        </QueryClientProvider>
      </NextThemesProvider>
    </NextUIProvider>
  );
}
