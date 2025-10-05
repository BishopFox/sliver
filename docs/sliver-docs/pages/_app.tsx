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
    </HeroUIProvider>
  );
}
