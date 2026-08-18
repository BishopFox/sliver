import MarkdownViewer from "@/components/markdown";
import LoadingState from "@/components/loading-state";
import { Tutorials } from "@/util/tutorials";
import { PREBUILD_VERSION } from "@/util/__generated__/prebuild-version";
import { fetchTutorials as fetchTutorialsContent } from "@/util/content-fetchers";
import {
  faChevronRight,
  faGraduationCap,
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

const TutorialsIndexPage: NextPage = () => {
  const router = useRouter();

  const { data: tutorials, isLoading } = useQuery({
    queryKey: ["tutorials", PREBUILD_VERSION],
    queryFn: () => fetchTutorialsContent(PREBUILD_VERSION),
  });

  const params = useSearchParams();
  const name = params.get("name") || "";
  const markdown = React.useMemo(() => {
    return tutorials?.tutorials.find((tutorial) => tutorial.name === name)?.content || "";
  }, [tutorials, name]);
  const hasNameQueryInPath = React.useMemo(() => {
    const query = router.asPath.split("?")[1];
    if (!query) {
      return false;
    }
    return new URLSearchParams(query).has("name");
  }, [router.asPath]);

  const [filterValue, setFilterValue] = React.useState("");
  const fuse = React.useMemo(() => {
    return new Fuse(tutorials?.tutorials || [], {
      keys: ["name"],
      threshold: 0.3,
    });
  }, [tutorials]);

  const visibleTutorials = React.useMemo(() => {
    if (filterValue) {
      // Fuzzy match display names
      const fuzzy = fuse.search(filterValue).map((r) => r.item);
      return fuzzy;
    }
    return tutorials?.tutorials || [];
  }, [tutorials, fuse, filterValue]);

  const mobileTutorials = React.useMemo(() => {
    if (!name) {
      return visibleTutorials;
    }
    if (visibleTutorials.some((tutorial) => tutorial.name === name)) {
      return visibleTutorials;
    }
    const selectedTutorial = tutorials?.tutorials.find((tutorial) => tutorial.name === name);
    return selectedTutorial ? [selectedTutorial, ...visibleTutorials] : visibleTutorials;
  }, [name, tutorials, visibleTutorials]);

  const listboxClasses =
    "flex flex-col gap-1 overflow-visible bg-transparent p-0";

  const renderFilterInput = (className?: string) => (
    <SearchField
      aria-label="Filter tutorials"
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
          <SearchField.ClearButton aria-label="Clear tutorial filter" />
        ) : null}
      </SearchField.Group>
    </SearchField>
  );

  if (isLoading || !tutorials || (hasNameQueryInPath && !name)) {
    return <LoadingState />;
  }

  return (
    <>
      <Head>
        <title>{name ? `${name} · Sliver Tutorials` : "Sliver Tutorials"}</title>
      </Head>
      <div className="mx-auto w-full max-w-[90rem] px-4 pt-8 sm:px-6 lg:px-8 lg:pt-10">
        <div className="mb-8 lg:hidden">
          <div className="flex items-center gap-3">
            <span className="flex size-9 items-center justify-center rounded-2xl bg-surface-secondary text-accent">
              <FontAwesomeIcon icon={faGraduationCap} />
            </span>
            <div>
              <p className="text-sm font-medium text-foreground">Tutorials</p>
              <p className="text-xs text-muted">Follow a guided workflow</p>
            </div>
          </div>
          <div className="mt-5">{renderFilterInput()}</div>
          <label
            htmlFor="tutorials-mobile-selector"
            className="mt-4 block text-sm font-medium text-foreground"
          >
            Tutorial
          </label>
          <select
            id="tutorials-mobile-selector"
            className="mt-2 w-full rounded-2xl border border-transparent bg-surface-secondary px-3 py-2.5 text-sm text-foreground outline-none focus:border-accent"
            value={name}
            onChange={(event) => {
              const selectedName = event.target.value;
              router.push({
                pathname: "/tutorials",
                query: selectedName ? { name: selectedName } : undefined,
              });
            }}
          >
            <option value="">Browse tutorials…</option>
            {mobileTutorials.map((tutorial) => (
              <option key={tutorial.name} value={tutorial.name}>
                {tutorial.name}
              </option>
            ))}
          </select>
        </div>

        <div className="grid min-w-0 grid-cols-1 gap-8 lg:grid-cols-[17rem_minmax(0,1fr)] lg:gap-10">
          <aside className="sliver-sticky-sidebar hidden border-r border-separator/70 pr-6 lg:block">
            <div className="flex h-full min-h-0 flex-col gap-5">
              <div className="flex items-center gap-3 px-1">
                <span className="flex size-9 items-center justify-center rounded-2xl bg-surface-secondary text-accent">
                  <FontAwesomeIcon icon={faGraduationCap} />
                </span>
                <div>
                  <p className="text-sm font-medium text-foreground">Tutorials</p>
                  <p className="text-xs text-muted">
                    {tutorials.tutorials.length} chapters
                  </p>
                </div>
              </div>
              {renderFilterInput()}
              <ScrollShadow className="sliver-scrollbar min-h-0 flex-1 overscroll-contain overflow-y-auto pr-2">
                <ListBox
                  aria-label="Tutorials"
                  className={listboxClasses}
                  selectedKeys={name ? [name] : []}
                  selectionMode="single"
                >
                  {visibleTutorials.map((tutorial) => (
                    <ListBox.Item
                      key={tutorial.name}
                      id={tutorial.name}
                      href={`/tutorials/?name=${encodeURIComponent(tutorial.name)}`}
                      textValue={tutorial.name}
                      className={`min-h-10 rounded-xl px-3 py-2 text-sm ${
                        tutorial.name === name
                          ? "bg-surface font-medium text-foreground shadow-surface"
                          : "text-muted"
                      }`}
                    >
                      {tutorial.name}
                    </ListBox.Item>
                  ))}
                </ListBox>
              </ScrollShadow>
            </div>
          </aside>

          <div
            key={name || "tutorials-overview"}
            className="min-w-0 pb-16 lg:pr-4"
          >
            {name !== "" ? (
              <article className="mx-auto w-full max-w-4xl">
                <header className="mb-8 border-b border-separator/70 pb-8">
                  <p className="text-sm font-medium text-accent">Tutorial</p>
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
                <h1 className="text-sm font-medium text-accent">
                  Guided learning
                </h1>
                <p className="mt-4 max-w-2xl text-lg leading-8 text-muted">
                  Follow practical walkthroughs covering setup, sessions,
                  staging, pivots, scripting, and post-exploitation workflows.
                </p>

                <div className="mt-10 grid gap-4 sm:grid-cols-2">
                  {tutorials.tutorials.slice(0, 6).map((tutorial) => (
                    <Card key={tutorial.name} className="h-full">
                      <Card.Header>
                        <Card.Title>{tutorial.name}</Card.Title>
                        <Card.Description>
                          Continue through this guided Sliver workflow.
                        </Card.Description>
                      </Card.Header>
                      <Card.Footer className="mt-auto">
                        <Button
                          variant="ghost"
                          onPress={() => {
                            router.push({
                              pathname: "/tutorials",
                              query: { name: tutorial.name },
                            });
                          }}
                        >
                          Start tutorial
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

export default TutorialsIndexPage;
