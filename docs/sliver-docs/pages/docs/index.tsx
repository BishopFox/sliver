import { Themes } from "@/util/themes";
import {
  Card,
  CardBody,
  CardHeader,
  Divider,
  Listbox,
  ListboxItem,
} from "@nextui-org/react";
import { useQuery } from "@tanstack/react-query";
import { NextPage } from "next";
import { useTheme } from "next-themes";
import React from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";

type Doc = {
  name: string;
  content: string;
};

type Docs = {
  docs: Doc[];
};

const DocsIndexPage: NextPage = () => {
  const { theme } = useTheme();

  const { data: docs, isLoading } = useQuery({
    queryKey: ["docs"],
    queryFn: async (): Promise<Docs> => {
      const res = await fetch("/docs.json");
      return res.json();
    },
  });

  const [name, setName] = React.useState(docs?.docs[0].name || "");
  const [markdown, setMarkdown] = React.useState(docs?.docs[0].content || "");

  React.useEffect(() => {
    if (docs) {
      setName(docs.docs[0].name);
      setMarkdown(docs.docs[0].content);
    }
  }, [docs]);

  if (isLoading || !docs) {
    return <div>Loading...</div>;
  }

  return (
    <div className="grid grid-cols-12">
      <div className="col-span-2 mt-6 ml-4">
        <div className="flex flex-row justify-center text-lg mb-2 gap-2">
          Topics
        </div>
        <div className="mt-2">
          <Listbox
            aria-label="Toolbox Menu"
            className="p-0 gap-0 divide-y divide-default-300/50 dark:divide-default-100/80 bg-content1 overflow-visible shadow-small rounded-medium"
            itemClasses={{
              base: "px-3 first:rounded-t-medium last:rounded-b-medium rounded-none gap-3 h-12 data-[hover=true]:bg-default-100/80",
            }}
          >
            {docs.docs.map((doc) => (
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
      </div>
      <div className="col-span-10">
        <Card className="mt-8 ml-8 mr-8 mb-8">
          <CardHeader>
            <span className="text-3xl">{name}</span>
          </CardHeader>
          <Divider />
          <CardBody
            className={
              theme === Themes.DARK ? "prose prose-invert" : "prose prose-slate"
            }
          >
            <Markdown remarkPlugins={[remarkGfm]}>{markdown}</Markdown>
          </CardBody>
        </Card>
      </div>
    </div>
  );
};

export default DocsIndexPage;
