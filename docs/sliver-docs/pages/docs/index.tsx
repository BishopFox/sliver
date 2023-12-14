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
  const { data, isLoading } = useQuery({
    queryKey: ["docs"],
    queryFn: async (): Promise<Docs> => {
      const res = await fetch("/docs.json");
      return res.json();
    },
  });

  const docsMap = React.useMemo(() => {
    if (!data) {
      return null;
    }
    const docsMap = new Map<string, string>();
    data.docs.forEach((doc) => {
      docsMap.set(doc.name, doc.content);
    });
    return docsMap;
  }, [data]);

  if (isLoading) {
    return <div>Loading...</div>;
  }

  return (
    <div>
      <h1>Docs Index</h1>
      <ul>
        {data?.docs.map((doc) => (
          <li key={doc.name}>
            <a href={`/docs/${doc.name}`}>{doc.name}</a>
          </li>
        ))}
      </ul>
    </div>
  );
};

export default DocsIndexPage;
