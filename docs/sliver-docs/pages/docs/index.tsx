import MarkdownViewer from "@/components/markdown";
import { Docs } from "@/util/docs";
import { frags } from "@/util/frags";
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
import React from "react";

const DocsIndexPage: NextPage = () => {
  const { theme } = useTheme();

  const { data: docs, isLoading } = useQuery({
    queryKey: ["docs"],
    queryFn: async (): Promise<Docs> => {
      const res = await fetch("/docs.json");
      return res.json();
    },
  });

  const [name, _setName] = React.useState(decodeURI(frags.get("name") || ""));
  const setName = (name: string) => {
    frags.set("name", name);
    _setName(name);
  };
  const [markdown, setMarkdown] = React.useState(
    name === ""
      ? docs?.docs[0].content
      : docs?.docs.find((doc) => doc.name === name)?.content || ""
  );
  React.useEffect(() => {
    if (docs && name !== "") {
      setMarkdown(docs?.docs.find((doc) => doc.name === name)?.content);
    }
    if (docs && name === "" && docs.docs.length > 0) {
      setName(docs.docs[0].name);
      setMarkdown(docs.docs[0].content);
    }
  }, [docs, name]);

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
      <div className="col-span-3 mt-4 ml-4 h-screen sticky top-20">
        <div className="flex flex-row justify-center text-lg gap-2">
          <Input
            label="Filter"
            isClearable={true}
            onClear={() => setFilterValue("")}
            placeholder="Type to Filter..."
            startContent={<FontAwesomeIcon icon={faSearch} />}
            value={filterValue}
            onChange={(e) => setFilterValue(e.target.value)}
          />
        </div>
        <div className="mt-2">
          <ScrollShadow>
            <div className="max-h-[80vh]">
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
                      setName(doc.name);
                      setMarkdown(doc.content);
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
        <Card className="mt-8 ml-8 mr-8 mb-8">
          <CardHeader>
            <span className="text-3xl">{name}</span>
          </CardHeader>
          <Divider />
          <CardBody>
            <MarkdownViewer markdown={markdown || ""} />
          </CardBody>
        </Card>
      </div>
    </div>
  );
};

export default DocsIndexPage;
