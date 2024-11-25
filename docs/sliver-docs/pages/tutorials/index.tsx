import MarkdownViewer from "@/components/markdown";
import { Tutorials } from "@/util/tutorials";
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
} from "@nextui-org/react";
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
    queryKey: ["tutorials"],
    queryFn: async (): Promise<Tutorials> => {
      const res = await fetch("/tutorials.json");
      return res.json();
    },
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

  const listboxClasses = React.useMemo(() => {
    if (theme === Themes.DARK) {
      return "p-0 gap-0 divide-y divide-default-300/50 dark:divide-default-100/80 bg-content1 overflow-visible shadow-small rounded-medium";
    } else {
      return "border p-0 gap-0 divide-y divide-default-300/50 dark:divide-default-100/80 bg-content1 overflow-visible shadow-small rounded-medium";
    }
  }, [theme]);

  if (isLoading || !tutorials) {
    return <div>Loading...</div>;
  }

  return (
    <div className="grid grid-cols-12">
      <Head>
        <title>Sliver Tutorial: {name}</title>
      </Head>
      <div className="col-span-3 mt-4 ml-4 h-screen sticky top-20">
        <div className="flex flex-row justify-center text-lg gap-2">
          <Input
            placeholder="Filter..."
            startContent={<FontAwesomeIcon icon={faSearch} />}
            value={filterValue}
            onChange={(e) => setFilterValue(e.target.value)}
            isClearable={true}
            onClear={() => setFilterValue("")}
          />
        </div>
        <div className="mt-2">
          <ScrollShadow>
            <div className="max-h-[70vh]">
              <Listbox
                aria-label="Toolbox Menu"
                className={listboxClasses}
                itemClasses={{
                  base: "px-3 first:rounded-t-medium last:rounded-b-medium rounded-none gap-3 h-12 data-[hover=true]:bg-default-100/80",
                }}
              >
                {visibleTutorials.map((tutorial) => (
                  <ListboxItem
                    key={tutorial.name}
                    value={tutorial.name}
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
            </div>
          </ScrollShadow>
        </div>
      </div>
      <div className="col-span-9">
        {name !== "" ? (
          <Card className="mt-8 ml-8 mr-8 mb-8">
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
          <div className="grid grid-cols-3">
            <div className="col-span-1"></div>
            <div className="col-span-1 mt-8 text-2xl text-center">
              Welcome to the Sliver Tutorials!
              <div className="text-xl text-gray-500">
                Please select a chapter
              </div>
            </div>
            <div className="col-span-1"></div>
          </div>
        )}
      </div>
    </div>
  );
};

export default TutorialsIndexPage;
