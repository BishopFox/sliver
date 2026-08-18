import MarkdownViewer from "@/components/markdown";
import LoadingState from "@/components/loading-state";
import { Docs } from "@/util/docs";
import { PREBUILD_VERSION } from "@/util/__generated__/prebuild-version";
import { fetchDocs as fetchDocsContent } from "@/util/content-fetchers";
import {
  faBookOpen,
  faChevronRight,
  faSearch,
} from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  Button,
  Card,
  ListBox,
  ScrollShadow,
  SearchField,
} from "@heroui/react";
import { useQuery } from "@tanstack/react-query";
import Fuse from "fuse.js";
import { NextPage } from "next";
import Head from "next/head";
import { useSearchParams } from "next/navigation";
import { useRouter } from "next/router";
import React from "react";

const featuredDocNames = [
  "Getting Started",
  "Compile from Source",
  "Architecture",
  "Armory",
  "Stagers",
  "Troubleshooting",
];

const DocsIndexPage: NextPage = () => {
  const router = useRouter();

  const { data: docs, isLoading } = useQuery({
    queryKey: ["docs", PREBUILD_VERSION],
    queryFn: () => fetchDocsContent(PREBUILD_VERSION),
  });

  const params = useSearchParams();
  const name = params.get("name") || "";
  const markdown = React.useMemo(() => {
    return docs?.docs.find((doc) => doc.name === name)?.content || "";
  }, [docs, name]);
  const hasNameQueryInPath = React.useMemo(() => {
    const query = router.asPath.split("?")[1];
    if (!query) {
      return false;
    }
    return new URLSearchParams(query).has("name");
  }, [router.asPath]);

  const [filterValue, setFilterValue] = React.useState("");
  const fuse = React.useMemo(() => {
    return new Fuse(docs?.docs || [], {
      keys: ["name"],
      threshold: 0.3,
    });
  }, [docs]);

  const visibleDocs = React.useMemo(() => {
    if (filterValue) {
      // Fuzzy match display names
      const fuzzy = fuse.search(filterValue).map((r) => r.item);
      return fuzzy;
    }
    return docs?.docs || [];
  }, [docs, fuse, filterValue]);

  const mobileDocs = React.useMemo(() => {
    if (!name) {
      return visibleDocs;
    }
    if (visibleDocs.some((doc) => doc.name === name)) {
      return visibleDocs;
    }
    const selectedDoc = docs?.docs.find((doc) => doc.name === name);
    return selectedDoc ? [selectedDoc, ...visibleDocs] : visibleDocs;
  }, [docs, name, visibleDocs]);

  const listboxClasses =
    "flex flex-col gap-1 overflow-visible bg-transparent p-0";

  const featuredDocs = React.useMemo(() => {
    return featuredDocNames
      .map((featuredName) => docs?.docs.find((doc) => doc.name === featuredName))
      .filter((doc) => doc !== undefined);
  }, [docs]);

  const renderFilterInput = (className?: string) => (
    <SearchField
      aria-label="Filter documents"
      className={className}
      fullWidth
      variant="secondary"
      value={filterValue}
      onChange={setFilterValue}
    >
      <SearchField.Group>
        <SearchField.SearchIcon>
          <FontAwesomeIcon icon={faSearch} />
        </SearchField.SearchIcon>
        <SearchField.Input placeholder="Filter..." />
        {filterValue.length > 0 ? (
          <SearchField.ClearButton aria-label="Clear document filter" />
        ) : null}
      </SearchField.Group>
    </SearchField>
  );

  if (isLoading || !docs || (hasNameQueryInPath && !name)) {
    return <LoadingState />;
  }

  return (
    <>
      <Head>
        <title>{name ? `${name} · Sliver Docs` : "Sliver Documentation"}</title>
      </Head>
      <div className="mx-auto w-full max-w-[90rem] px-4 pt-8 sm:px-6 lg:px-8 lg:pt-10">
        <div className="mb-8 lg:hidden">
          <div className="flex items-center gap-3">
            <span className="flex size-9 items-center justify-center rounded-2xl bg-surface-secondary text-accent">
              <FontAwesomeIcon icon={faBookOpen} />
            </span>
            <div>
              <p className="text-sm font-medium text-foreground">Documentation</p>
              <p className="text-xs text-muted">Browse the reference library</p>
            </div>
          </div>
          <div className="mt-5">{renderFilterInput()}</div>
          <label
            htmlFor="docs-mobile-selector"
            className="mt-4 block text-sm font-medium text-foreground"
          >
            Document
          </label>
          <select
            id="docs-mobile-selector"
            className="mt-2 w-full rounded-2xl border border-transparent bg-surface-secondary px-3 py-2.5 text-sm text-foreground outline-none focus:border-accent"
            value={name}
            onChange={(event) => {
              const selectedName = event.target.value;
              router.push({
                pathname: "/docs",
                query: selectedName ? { name: selectedName } : undefined,
              });
            }}
          >
            <option value="">Browse documents…</option>
            {mobileDocs.map((doc) => (
              <option key={doc.name} value={doc.name}>
                {doc.name}
              </option>
            ))}
          </select>
        </div>

        <div className="grid min-w-0 grid-cols-1 gap-8 lg:grid-cols-[17rem_minmax(0,1fr)] lg:gap-10">
          <aside className="sliver-sticky-sidebar hidden border-r border-separator/70 pr-6 lg:block">
            <div className="flex h-full min-h-0 flex-col gap-5">
              <div className="flex items-center gap-3 px-1">
                <span className="flex size-9 items-center justify-center rounded-2xl bg-surface-secondary text-accent">
                  <FontAwesomeIcon icon={faBookOpen} />
                </span>
                <div>
                  <p className="text-sm font-medium text-foreground">Documentation</p>
                  <p className="text-xs text-muted">{docs.docs.length} articles</p>
                </div>
              </div>
              {renderFilterInput()}
              <ScrollShadow className="sliver-scrollbar min-h-0 flex-1 overscroll-contain overflow-y-auto pr-2">
                <ListBox
                  aria-label="Documentation"
                  className={listboxClasses}
                  selectedKeys={name ? [name] : []}
                  selectionMode="single"
                >
                  {visibleDocs.map((doc) => (
                    <ListBox.Item
                      key={doc.name}
                      id={doc.name}
                      href={`/docs/?name=${encodeURIComponent(doc.name)}`}
                      textValue={doc.name}
                      className={`min-h-10 rounded-xl px-3 py-2 text-sm ${
                        doc.name === name
                          ? "bg-surface font-medium text-foreground shadow-surface"
                          : "text-muted"
                      }`}
                    >
                      {doc.name}
                    </ListBox.Item>
                  ))}
                </ListBox>
              </ScrollShadow>
            </div>
          </aside>

          <div
            key={name || "documentation-overview"}
            className="min-w-0 pb-16 lg:pr-4"
          >
            {name !== "" ? (
              <article className="mx-auto w-full max-w-4xl">
                <header className="mb-8 border-b border-separator/70 pb-8">
                  <p className="text-sm font-medium text-accent">Reference</p>
                  <h1 className="mt-2 text-balance text-4xl font-semibold tracking-tight text-foreground sm:text-5xl">
                    {name}
                  </h1>
                </header>
                <MarkdownViewer
                  key={name}
                  markdown={markdown || ""}
                  demoteTopLevelHeading
                />
              </article>
            ) : (
              <div className="mx-auto w-full max-w-5xl pt-4 lg:pt-8">
                <p className="text-sm font-medium text-accent">Sliver reference</p>
                <h1 className="mt-2 max-w-3xl text-balance text-4xl font-semibold tracking-tight text-foreground sm:text-5xl">
                  Find the answer, then get back to the operation.
                </h1>
                <p className="mt-4 max-w-2xl text-lg leading-8 text-muted">
                  Browse configuration, transports, extensions, payloads, and
                  troubleshooting guidance for the Sliver framework.
                </p>

                <div className="mt-10 grid gap-4 sm:grid-cols-2">
                  {featuredDocs.map((doc) => (
                    <Card key={doc.name} className="h-full">
                      <Card.Header>
                        <Card.Title>{doc.name}</Card.Title>
                        <Card.Description>
                          Open this topic in the Sliver reference.
                        </Card.Description>
                      </Card.Header>
                      <Card.Footer className="mt-auto">
                        <Button
                          variant="ghost"
                          onPress={() => {
                            router.push({
                              pathname: "/docs",
                              query: { name: doc.name },
                            });
                          }}
                        >
                          Open article
                          <FontAwesomeIcon icon={faChevronRight} />
                        </Button>
                      </Card.Footer>
                    </Card>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </>
  );
};

export default DocsIndexPage;
