import MarkdownViewer from "@/components/markdown";
import { Docs } from "@/util/docs";
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

const DocsIndexPage: NextPage = () => {
  const { theme } = useTheme();
  const router = useRouter();

  const { data: docs, isLoading } = useQuery({
    queryKey: ["docs"],
    queryFn: async (): Promise<Docs> => {
      const res = await fetch("/docs.json");
      return res.json();
    },
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

  const listboxClasses = React.useMemo(() => {
    if (theme === Themes.DARK) {
      return "p-0 gap-0 divide-y divide-default-300/50 dark:divide-default-100/80 bg-content1 overflow-visible shadow-small rounded-medium";
    } else {
      return "border p-0 gap-0 divide-y divide-default-300/50 dark:divide-default-100/80 bg-content1 overflow-visible shadow-small rounded-medium";
    }
  }, [theme]);

  if (isLoading || !docs) {
    return <div>Loading...</div>;
  }

  return (
    <div className="grid grid-cols-12">
      <Head>
        <title>Sliver Docs: {name}</title>
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
                {visibleDocs.map((doc) => (
                  <ListboxItem
                    key={doc.name}
                    value={doc.name}
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
              Welcome to the Sliver Wiki!
              <div className="text-xl text-gray-500">
                Please select a document
              </div>
            </div>
            <div className="col-span-1"></div>
          </div>
        )}
      </div>
    </div>
  );
};

export default DocsIndexPage;
