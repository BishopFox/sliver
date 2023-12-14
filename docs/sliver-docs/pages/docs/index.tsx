import CodeViewer, { CodeSchema } from "@/components/code";
import { useQuery } from "@tanstack/react-query";
import { NextPage } from "next";
import React from "react";

type Doc = {
  name: string;
  content: string;
};

type Docs = {
  docs: Doc[];
};

const DocsIndexPage: NextPage = () => {
  const { data: docs, isLoading } = useQuery({
    queryKey: ["docs"],
    queryFn: async (): Promise<Docs> => {
      const res = await fetch("/docs.json");
      return res.json();
    },
  });

  const [markdown, setMarkdown] = React.useState(docs?.docs[0].content || "");

  React.useEffect(() => {
    if (docs) {
      setMarkdown(docs.docs[0].content);
    }
  }, [docs]);

  if (isLoading || !docs) {
    return <div>Loading...</div>;
  }

  return (
    <div className="grid grid-cols-12">
      <div className="col-span-1"></div>
      <div className="col-span-10">
        <h1>Docs Index</h1>
        <div>
          <CodeViewer
            className="min-h-[150px]"
            script={
              {
                name: "",
                script_type: "plaintext",
                source_code: "asf",
              } as CodeSchema
            }
          />
        </div>
        <div></div>
      </div>
      <div className="col-span-1"></div>
    </div>
  );
};

export default DocsIndexPage;
