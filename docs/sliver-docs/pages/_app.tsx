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
import { RouterProvider } from "@heroui/react";
import {
  HydrationBoundary,
  QueryClient,
  QueryClientProvider,
} from "@tanstack/react-query";
import { ThemeProvider as NextThemesProvider } from "next-themes";
import type { AppProps } from "next/app";
import { useRouter } from "next/router";
import React from "react";

export default function App({ Component, pageProps }: AppProps) {
  const router = useRouter();

  const navigate = React.useCallback(
    (path: string) => {
      void router.push(path);
    },
    [router],
  );

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
    <NextThemesProvider
      attribute="class"
      defaultTheme={Themes.DARK}
      enableSystem={false}
      storageKey="theme"
    >
      <QueryClientProvider client={queryClient}>
        <HydrationBoundary state={pageProps.dehydratedState}>
          <RouterProvider navigate={navigate}>
            <SearchContext.Provider value={search}>
              <div className="flex min-h-dvh flex-col">
                <Navbar />
                <main className="sliver-page-content flex-1">
                  <Component {...pageProps} />
                </main>
                <footer className="sliver-fixed-footer fixed inset-x-0 bottom-0 z-30 border-t border-separator/70 bg-background/70 backdrop-blur-xl">
                  <div className="mx-auto flex min-h-[var(--sliver-footer-height)] w-full max-w-7xl flex-col items-center justify-center gap-1.5 px-4 py-3 text-center text-sm text-muted sm:px-6 md:flex-row md:justify-between md:text-left lg:px-8">
                    <span>
                      © {new Date().getFullYear()} {" "}
                      <a
                        href="https://bishopfox.com/"
                        className="font-medium text-foreground no-underline hover:text-accent"
                        rel="noreferrer"
                        target="_blank"
                      >
                        Bishop Fox
                      </a>
                    </span>
                    <div className="flex flex-wrap items-center justify-center gap-x-5 gap-y-2">
                      <a
                        href="https://github.com/BishopFox/sliver/pulls"
                        rel="noreferrer"
                        className="inline-flex items-center gap-1.5 no-underline hover:text-foreground"
                        target="_blank"
                      >
                        Improve these docs
                        <FontAwesomeIcon className="text-xs" icon={faExternalLink} />
                      </a>
                      <a
                        href="https://github.com/BishopFox/sliver/blob/master/LICENSE"
                        target="_blank"
                        rel="noreferrer"
                        className="no-underline hover:text-foreground"
                      >
                        GPLv3
                      </a>
                    </div>
                  </div>
                </footer>
              </div>
            </SearchContext.Provider>
          </RouterProvider>
        </HydrationBoundary>
      </QueryClientProvider>
    </NextThemesProvider>
  );
}
