import Navbar from "@/components/navbar";
import "@/styles/globals.css";
import { Docs } from "@/util/docs";
import { PREBUILD_VERSION } from "@/util/__generated__/prebuild-version";
import { Tutorials } from "@/util/tutorials";
import { SearchContext, SearchCtx } from "@/util/search-context";
import { fetchDocs as fetchDocsContent, fetchTutorials as fetchTutorialsContent } from "@/util/content-fetchers";
import { Themes } from "@/util/themes";
import { faExternalLink } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { HeroUIProvider } from "@heroui/react";
import {
  HydrationBoundary,
  QueryClient,
  QueryClientProvider,
} from "@tanstack/react-query";
import { ThemeProvider as NextThemesProvider } from "next-themes";
import type { AppProps } from "next/app";
import React from "react";

export default function App({ Component, pageProps }: AppProps) {
  // Initialize search
  const [search] = React.useState(() => new SearchCtx());

  // Initialize query client
  const [queryClient] = React.useState(() => new QueryClient());
  const versionRef = React.useRef(PREBUILD_VERSION);

  React.useEffect(() => {
    const docsFetcher = () => fetchDocsContent(versionRef.current);
    const tutorialsFetcher = () => fetchTutorialsContent(versionRef.current);

    queryClient.setQueryDefaults(["docs"], {
      queryFn: docsFetcher,
    });
    queryClient.setQueryDefaults(["tutorials"], {
      queryFn: tutorialsFetcher,
    });
  }, [queryClient]);

  React.useEffect(() => {
    const syncSearch = () => {
      const docsQueries = queryClient.getQueriesData<Docs>({ queryKey: ["docs"] });
      docsQueries.forEach(([, data]) => {
        if (data) {
          search.addDocs(data);
        }
      });

      const tutorialQueries = queryClient.getQueriesData<Tutorials>({ queryKey: ["tutorials"] });
      tutorialQueries.forEach(([, data]) => {
        if (data) {
          search.addTutorials(data);
        }
      });
    };

    syncSearch();
    const unsubscribe = queryClient.getQueryCache().subscribe(syncSearch);
    return unsubscribe;
  }, [queryClient, search]);

  React.useEffect(() => {
    const docsFetcher = () => fetchDocsContent(versionRef.current);
    const tutorialsFetcher = () => fetchTutorialsContent(versionRef.current);

    void queryClient.prefetchQuery({
      queryKey: ["docs", versionRef.current],
      queryFn: docsFetcher,
    });
    void queryClient.prefetchQuery({
      queryKey: ["tutorials", versionRef.current],
      queryFn: tutorialsFetcher,
    });
  }, [queryClient]);

  React.useEffect(() => {
    if (versionRef.current !== PREBUILD_VERSION) {
      versionRef.current = PREBUILD_VERSION;
      queryClient.invalidateQueries({ queryKey: ["docs"] });
      queryClient.invalidateQueries({ queryKey: ["tutorials"] });

      const docsFetcher = () => fetchDocsContent(versionRef.current);
      const tutorialsFetcher = () => fetchTutorialsContent(versionRef.current);
      void queryClient.prefetchQuery({
        queryKey: ["docs", versionRef.current],
        queryFn: docsFetcher,
      });
      void queryClient.prefetchQuery({
        queryKey: ["tutorials", versionRef.current],
        queryFn: tutorialsFetcher,
      });
    }
  });

  return (
    <HeroUIProvider>
      <NextThemesProvider
        attribute="class"
        defaultTheme={Themes.DARK}
        enableSystem={false}
        storageKey="theme"
      >
        <QueryClientProvider client={queryClient}>
          <HydrationBoundary state={pageProps.dehydratedState}>
            <SearchContext.Provider value={search}>
              <Navbar />
              <Component {...pageProps} />
              <div className="mb-16 md:mb-20"></div>
              <footer className="z-20 w-full border-t border-gray-200 bg-white px-3 py-4 text-center shadow dark:bg-black dark:border-gray-600 md:fixed md:bottom-0 md:left-0 md:flex md:items-center md:justify-between md:px-6 md:py-4">
                <div className="flex flex-col items-center gap-2 text-xs leading-snug text-gray-500 sm:text-sm md:flex-row md:items-center md:gap-3 md:text-left dark:text-gray-400">
                  <span>
                    © {new Date().getFullYear()}&nbsp;
                    <a
                      href="https://bishopfox.com/"
                      className="hover:underline"
                      rel="noreferrer"
                      target="_blank"
                    >
                      Bishop Fox
                    </a>
                  </span>
                  <span className="hidden md:inline" aria-hidden="true">
                    ·
                  </span>
                  <span className="max-w-xs md:max-w-none">
                    <a
                      href="https://github.com/BishopFox/sliver/pulls"
                      rel="noreferrer"
                      className="hover:underline"
                      target="_blank"
                    >
                      Help improve this documentation
                      <span className="ml-1 inline-block">
                        <FontAwesomeIcon icon={faExternalLink} />
                      </span>
                    </a>
                  </span>
                </div>
                <ul className="mt-3 flex flex-wrap items-center justify-center gap-3 text-xs font-medium text-gray-500 md:mt-0 md:gap-6 md:text-sm dark:text-gray-400">
                  <li>
                    <a
                      href="https://github.com/BishopFox/sliver/blob/master/LICENSE"
                      target="_blank"
                      rel="noreferrer"
                      className="hover:underline"
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
    </HeroUIProvider>
  );
}
