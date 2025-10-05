import MarkdownViewer from "@/components/markdown";
import { Docs } from "@/util/docs";
import { PREBUILD_VERSION } from "@/util/__generated__/prebuild-version";
import { fetchDocs as fetchDocsContent } from "@/util/content-fetchers";
import { Themes } from "@/util/themes";
import { faSearch } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  Card,
  CardBody,
  CardHeader,
  Divider,
  Input,
  Listbox,
  ListboxItem,
  ScrollShadow,
} from "@heroui/react";
import { useQuery } from "@tanstack/react-query";
import Fuse from "fuse.js";
import { NextPage } from "next";
import { useTheme } from "next-themes";
import Head from "next/head";
import { useSearchParams } from "next/navigation";
import { useRouter } from "next/router";
import React from "react";

const DocsIndexPage: NextPage = () => {
  const { theme } = useTheme();
  const router = useRouter();

  const { data: docs, isLoading } = useQuery({
    queryKey: ["docs", PREBUILD_VERSION],
    queryFn: () => fetchDocsContent(PREBUILD_VERSION),
  });

  const params = useSearchParams();
  const [name, setName] = React.useState("");
  const [markdown, setMarkdown] = React.useState("");

  React.useEffect(() => {
    const _name = params.get("name");
    setName(_name || "");
    setMarkdown(docs?.docs.find((doc) => doc.name === _name)?.content || "");
  }, [params, docs]);

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

  const listboxClasses = React.useMemo(() => {
    if (theme === Themes.DARK) {
      return "p-0 gap-0 divide-y divide-default-300/50 dark:divide-default-100/80 bg-content1 overflow-visible shadow-small rounded-large";
    } else {
      return "p-0 gap-0 divide-y divide-default-300/50 dark:divide-default-100/80 bg-content1 overflow-visible rounded-large";
    }
  }, [theme]);

  if (isLoading || !docs) {
    return <div>Loading...</div>;
  }

  return (
    <>
      <Head>
        <title>Sliver Docs: {name}</title>
      </Head>
      <div className="px-4 pt-4 lg:hidden">
        <label
          htmlFor="docs-mobile-selector"
          className="block text-sm font-medium text-foreground"
        >
          Select a document
        </label>
        <div className="mt-2">
          <Input
            placeholder="Filter..."
            startContent={<FontAwesomeIcon icon={faSearch} />}
            value={filterValue}
            onChange={(e) => setFilterValue(e.target.value)}
            isClearable={true}
            onClear={() => setFilterValue("")}
          />
        </div>
        <select
          id="docs-mobile-selector"
          className="mt-3 w-full rounded-lg border border-default-200 bg-content1 p-2 text-sm dark:border-default-100/60"
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
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-12 lg:gap-8">
        <aside className="hidden lg:block lg:col-span-3">
          <div className="sticky top-24 ml-4 flex flex-col gap-3">
            <Input
            className="mt-2"
              placeholder="Filter..."
              startContent={<FontAwesomeIcon icon={faSearch} />}
              value={filterValue}
              onChange={(e) => setFilterValue(e.target.value)}
              isClearable={true}
              onClear={() => setFilterValue("")}
            />
            <ScrollShadow className="max-h-[calc(100vh-15rem)] sliver-scrollbar overflow-y-auto pr-1 rounded-large">
              <Listbox
                aria-label="Toolbox Menu"
                className={listboxClasses}
                itemClasses={{
                  base: "px-3 first:rounded-t-large last:rounded-b-large rounded-none gap-3 h-12 data-[hover=true]:bg-default-100/80",
                }}
              >
                {visibleDocs.map((doc) => (
                  <ListboxItem
                    key={doc.name}
                    onClick={() => {
                      router.push({
                        pathname: "/docs",
                        query: { name: doc.name },
                      });
                    }}
                  >
                    {doc.name}
                  </ListboxItem>
                ))}
              </Listbox>
            </ScrollShadow>
          </div>
        </aside>
        <div className="px-4 pb-8 lg:col-span-9 lg:px-8">
          {name !== "" ? (
            <Card className="mt-2">
              <CardHeader>
                <span className="text-3xl">{name}</span>
              </CardHeader>
              <Divider />
              <CardBody>
                <MarkdownViewer
                  key={name || `${Math.random()}`}
                  markdown={markdown || ""}
                />
              </CardBody>
            </Card>
          ) : (
            <div className="mt-8 text-center text-2xl text-foreground/90">
              Welcome to the Sliver Wiki!
              <div className="mt-2 text-xl text-default-500">
                Please select a document
              </div>
            </div>
          )}
        </div>
      </div>
    </>
  );
};

export default DocsIndexPage;
