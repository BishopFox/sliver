import Navbar from "@/components/navbar";
import "@/styles/globals.css";
import { Docs } from "@/util/docs";
import { Tutorials } from "@/util/tutorials";
import { SearchContext, SearchCtx } from "@/util/search-context";
import { Themes } from "@/util/themes";
import { faExternalLink } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
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

  queryClient.prefetchQuery({
    queryKey: ["tutorials"],
    queryFn: async (): Promise<Tutorials> => {
      const res = await fetch("/tutorials.json");
      const tutorials: Tutorials = await res.json();
      search.addTutorials(tutorials);
      return tutorials;
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
              <div className="mb-12"></div>
              <footer className="fixed bottom-0 left-0 z-20 w-full p-2 bg-white border-t border-gray-200 shadow md:flex md:items-center md:justify-between md:p-6 dark:bg-black dark:border-gray-600">
                <span className="text-sm text-gray-500 sm:text-center dark:text-gray-400">
                  © {new Date().getFullYear()}{" "}
                  <a
                    href="https://bishopfox.com/"
                    className="hover:underline"
                    rel="noreferrer"
                    target="_blank"
                  >
                    Bishop Fox
                  </a>
                  {" - "}
                  <a
                    href="https://github.com/BishopFox/sliver/pulls"
                    rel="noreferrer"
                    className="hover:underline"
                    target="_blank"
                  >
                    You can help improve this documentation by opening a pull
                    request on Github <FontAwesomeIcon icon={faExternalLink} />
                  </a>
                </span>
                <ul className="flex flex-wrap items-center mt-3 text-sm font-medium text-gray-500 dark:text-gray-400 sm:mt-0">
                  <li>
                    <a
                      href="https://github.com/BishopFox/sliver/blob/master/LICENSE"
                      target="_blank"
                      rel="noreferrer"
                      className="hover:underline me-4 md:me-6"
                    >
                      GPLv3 License
                    </a>
                  </li>
                </ul>
              </footer>
            </SearchContext.Provider>
          </HydrationBoundary>
        </QueryClientProvider>
      </NextThemesProvider>
    </NextUIProvider>
  );
}
