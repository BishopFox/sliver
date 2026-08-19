import LoadingState from "@/components/loading-state";
import { PREBUILD_VERSION } from "@/util/__generated__/prebuild-version";
import { fetchDocs as fetchDocsContent } from "@/util/content-fetchers";
import { SearchCtx } from "@/util/search-context";
import {
  faChevronRight,
  faMagnifyingGlass,
} from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { Button, Card, SearchField } from "@heroui/react";
import { useQuery } from "@tanstack/react-query";
import { NextPage } from "next";
import Head from "next/head";
import { useSearchParams } from "next/navigation";
import { useRouter } from "next/router";
import React from "react";

export type SearchPageProps = {};

const createExcerpt = (markdown: string) => {
  return markdown
    .replace(/```[\s\S]*?```/g, " ")
    .replace(/!\[[^\]]*\]\([^)]*\)/g, " ")
    .replace(/\[([^\]]+)\]\([^)]*\)/g, "$1")
    .replace(/[#>*_`~|-]/g, " ")
    .replace(/\s+/g, " ")
    .trim()
    .slice(0, 240);
};

const SearchPage: NextPage = (props: SearchPageProps) => {
  const router = useRouter();
  const query = useSearchParams().get("search")?.trim() || "";
  const { data: docs, isLoading } = useQuery({
    queryKey: ["docs", PREBUILD_VERSION],
    queryFn: () => fetchDocsContent(PREBUILD_VERSION),
  });

  const docsSearch = React.useMemo(() => {
    const search = new SearchCtx();
    if (docs) {
      search.addDocs(docs);
    }
    return search;
  }, [docs]);

  const searchResults = React.useMemo(() => {
    if (query) {
      return docsSearch.searchDocs(query);
    }
    return [];
  }, [docsSearch, query]);

  if (isLoading || !docs) {
    return <LoadingState />;
  }

  return (
    <>
      <Head>
        <title>{query ? `Search: ${query} · Sliver Docs` : "Search · Sliver Docs"}</title>
      </Head>
      <div className="mx-auto w-full max-w-5xl px-4 pb-20 pt-12 sm:px-6 lg:px-8 lg:pt-16">
        <header className="max-w-3xl">
          <p className="text-sm font-medium text-accent">Search the reference</p>
          <h1 className="mt-2 text-balance text-4xl font-semibold tracking-tight text-foreground sm:text-5xl">
            Find exactly what you need.
          </h1>
          <p className="mt-4 text-lg leading-8 text-muted">
            Search configuration, workflows, transports, extensions, and
            troubleshooting notes across the Sliver documentation.
          </p>
        </header>

        <SearchField
          key={query}
          aria-label="Search documentation"
          className="mt-8 max-w-2xl"
          defaultValue={query}
          fullWidth
          onClear={() => router.push("/search")}
          onSubmit={(value) => {
            const nextQuery = value.trim();
            if (nextQuery) {
              router.push({ pathname: "/search", query: { search: nextQuery } });
            }
          }}
        >
          <SearchField.Group>
            <SearchField.SearchIcon>
              <FontAwesomeIcon icon={faMagnifyingGlass} />
            </SearchField.SearchIcon>
            <SearchField.Input placeholder="Search documentation…" />
            <SearchField.ClearButton aria-label="Clear search" />
          </SearchField.Group>
        </SearchField>

        <div className="mt-10 flex items-end justify-between gap-4">
          <div>
            <h2 className="text-2xl font-semibold text-foreground">
              {query ? `Results for “${query.slice(0, 50)}”` : "Search results"}
            </h2>
            <p className="mt-1 text-sm tabular-nums text-muted">
              {searchResults.length} {searchResults.length === 1 ? "result" : "results"}
            </p>
          </div>
        </div>

        {searchResults.length > 0 ? (
          <div className="mt-5 grid gap-4">
            {searchResults.map((doc) => (
              <Card key={doc.name}>
                <Card.Header>
                  <Card.Title>{doc.name}</Card.Title>
                  <Card.Description className="mt-1 max-w-3xl leading-6">
                    {createExcerpt(doc.content) || "Open this article in the Sliver reference."}
                    {doc.content.length > 240 ? "…" : ""}
                  </Card.Description>
                </Card.Header>
                <Card.Footer>
                  <Button
                    variant="ghost"
                    onPress={() => {
                      router.push({ pathname: "/docs", query: { name: doc.name } });
                    }}
                  >
                    Open article
                    <FontAwesomeIcon icon={faChevronRight} />
                  </Button>
                </Card.Footer>
              </Card>
            ))}
          </div>
        ) : (
          <Card className="mt-5">
            <Card.Content className="flex flex-col items-start gap-3 p-8">
              <span className="flex size-10 items-center justify-center rounded-2xl bg-surface-secondary text-accent">
                <FontAwesomeIcon icon={faMagnifyingGlass} />
              </span>
              <div>
                <h2 className="text-lg font-semibold text-foreground">
                  {query ? "No matching documentation" : "Start with a keyword"}
                </h2>
                <p className="mt-1 max-w-xl text-sm leading-6 text-muted">
                  {query
                    ? "Try a command name, transport, feature, or a shorter phrase."
                    : "Search for topics like staging, WireGuard, profiles, or extensions."}
                </p>
              </div>
              {query ? (
                <Button
                  variant="outline"
                  onPress={() => router.push("/docs")}
                >
                  Browse all documentation
                </Button>
              ) : null}
            </Card.Content>
          </Card>
        )}
      </div>
    </>
  );
};

export default SearchPage;
