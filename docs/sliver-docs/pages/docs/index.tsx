import CodeViewer, { CodeSchema } from "@/components/code";
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
} from "@nextui-org/react";
import { useQuery } from "@tanstack/react-query";
import Fuse from "fuse.js";
import { NextPage } from "next";
import { useTheme } from "next-themes";
import Image from "next/image";
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

  if (isLoading || !docs) {
    return <div>Loading...</div>;
  }

  return (
    <div className="grid grid-cols-12">
      <div className="col-span-3 mt-4 ml-4">
        <div className="flex flex-row justify-center text-lg mb-2 gap-2">
          <Input
            label="Filter"
            isClearable={true}
            placeholder="Type to filter..."
            startContent={<FontAwesomeIcon icon={faSearch} />}
            value={filterValue}
            onChange={(e) => setFilterValue(e.target.value)}
          />
        </div>
        <div className="mt-2">
          <Listbox
            aria-label="Toolbox Menu"
            className="p-0 gap-0 divide-y divide-default-300/50 dark:divide-default-100/80 bg-content1 overflow-visible shadow-small rounded-medium"
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
      </div>
      <div className="col-span-9">
        <Card className="mt-8 ml-8 mr-8 mb-8">
          <CardHeader>
            <span className="text-3xl">{name}</span>
          </CardHeader>
          <Divider />
          <CardBody
            className={
              theme === Themes.DARK
                ? "prose prose-sm dark:prose-invert"
                : "prose prose-sm prose-slate"
            }
          >
            <Markdown
              remarkPlugins={[remarkGfm]}
              components={{
                pre(props) {
                  const { children, className, node, ...rest } = props;
                  const childClass = (children as any)?.props?.className;
                  if (
                    childClass &&
                    childClass.startsWith("language-") &&
                    childClass !== "language-plaintext"
                  ) {
                    // @ts-ignore
                    return <div {...rest}>{children}</div>;
                  }
                  return (
                    <pre {...rest} className={className}>
                      {children}
                    </pre>
                  );
                },

                img(props) {
                  const { src, alt, ...rest } = props;
                  return (
                    // @ts-ignore
                    <Image
                      {...rest}
                      src={src || ""}
                      alt={alt || ""}
                      width={500}
                      height={500}
                      className="w-full rounded-medium"
                    />
                  );
                },

                code(props) {
                  const { children, className, node, ...rest } = props;
                  const langTag = /language-(\w+)/.exec(className || "");
                  const lang = langTag ? langTag[1] : "plaintext";
                  if (lang === "plaintext") {
                    return (
                      <code {...rest} className={className}>
                        {children}
                      </code>
                    );
                  }
                  return (
                    <CodeViewer
                      className="min-h-[250px]"
                      key={`${Math.random()}`}
                      fontSize={11}
                      script={
                        {
                          script_type: lang,
                          source_code: (children as string) || "",
                        } as CodeSchema
                      }
                    />
                  );
                },
              }}
            >
              {markdown}
            </Markdown>
          </CardBody>
        </Card>
      </div>
    </div>
  );
};

export default DocsIndexPage;
