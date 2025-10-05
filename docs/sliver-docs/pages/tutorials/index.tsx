import MarkdownViewer from "@/components/markdown";
import { Tutorials } from "@/util/tutorials";
import { PREBUILD_VERSION } from "@/util/__generated__/prebuild-version";
import { fetchTutorials as fetchTutorialsContent } from "@/util/content-fetchers";
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

const TutorialsIndexPage: NextPage = () => {
  const { theme } = useTheme();
  const router = useRouter();

  const { data: tutorials, isLoading } = useQuery({
    queryKey: ["tutorials", PREBUILD_VERSION],
    queryFn: () => fetchTutorialsContent(PREBUILD_VERSION),
  });

  const params = useSearchParams();
  const [name, setName] = React.useState("");
  const [markdown, setMarkdown] = React.useState("");

  React.useEffect(() => {
    const _name = params.get("name");
    setName(_name || "");
    setMarkdown(tutorials?.tutorials.find((tutorial) => tutorial.name === _name)?.content || "");
  }, [params, tutorials]);

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

  const listboxClasses = React.useMemo(() => {
    if (theme === Themes.DARK) {
      return "p-0 gap-0 divide-y divide-default-300/50 dark:divide-default-100/80 bg-content1 overflow-visible shadow-small rounded-large";
    } else {
      return "p-0 gap-0 divide-y divide-default-300/50 dark:divide-default-100/80 bg-content1 overflow-visible rounded-large";
    }
  }, [theme]);

  if (isLoading || !tutorials) {
    return <div>Loading...</div>;
  }

  return (
    <>
      <Head>
        <title>Sliver Tutorial: {name}</title>
      </Head>
      <div className="px-4 pt-4 lg:hidden">
        <label
          htmlFor="tutorials-mobile-selector"
          className="block text-sm font-medium text-foreground"
        >
          Select a tutorial
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
          id="tutorials-mobile-selector"
          className="mt-3 w-full rounded-lg border border-default-200 bg-content1 p-2 text-sm dark:border-default-100/60"
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
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-12 lg:gap-8">
        <aside className="hidden lg:block lg:col-span-3">
          <div className="sticky top-24 ml-4 flex flex-col gap-3">
            <Input
              placeholder="Filter..."
              startContent={<FontAwesomeIcon icon={faSearch} />}
              value={filterValue}
              onChange={(e) => setFilterValue(e.target.value)}
              isClearable={true}
              onClear={() => setFilterValue("")}
            />
            <ScrollShadow className="max-h-[calc(100vh-6rem)] sliver-scrollbar overflow-y-auto pr-1 rounded-large">
              <Listbox
                aria-label="Toolbox Menu"
                className={listboxClasses}
                itemClasses={{
                  base: "px-3 first:rounded-t-large last:rounded-b-large rounded-none gap-3 h-12 data-[hover=true]:bg-default-100/80",
                }}
              >
                {visibleTutorials.map((tutorial) => (
                  <ListboxItem
                    key={tutorial.name}
                    onClick={() => {
                      router.push({
                        pathname: "/tutorials",
                        query: { name: tutorial.name },
                      });
                    }}
                  >
                    {tutorial.name}
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
              Welcome to the Sliver Tutorials!
              <div className="mt-2 text-xl text-default-500">
                Please select a chapter
              </div>
            </div>
          )}
        </div>
      </div>
    </>
  );
};

export default TutorialsIndexPage;
