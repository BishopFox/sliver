import Youtube from "@/components/youtube";
import { faChevronRight } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { Button, Card, CardFooter, CardHeader } from "@heroui/react";
import { NextPage } from "next";
import Head from "next/head";
import React from "react";

type Talk = {
  title: string;
  description: string;
  url: string;
};

const talks: Talk[] = [
  {
    title: "Building Traffic Encoders",
    description: "From concept to practical encoder strategy in Sliver.",
    url: "https://www.youtube.com/watch?v=6unwFhurm-E",
  },
  {
    title: "Sliver Staging and Automation",
    description: "Workflow patterns for payload staging and repeatable ops.",
    url: "https://www.youtube.com/watch?v=vuQ5tG5kelI&feature=youtu.be",
  },
  {
    title: "Offensive WASM",
    description: "Applying WebAssembly techniques in offensive tradecraft.",
    url: "https://www.youtube.com/watch?v=RnSLsnP4imQ",
  },
];

const TalksIndexPage: NextPage = () => {
  const [footerHidden, setFooterHidden] = React.useState<Record<string, boolean>>(
    {}
  );

  const hideFooterFor = React.useCallback((url: string) => {
    setFooterHidden((prev) => (prev[url] ? prev : { ...prev, [url]: true }));
  }, []);

  return (
    <>
      <Head>
        <title>Sliver Talks</title>
      </Head>
      <div className="mt-6 px-4 pb-8 sm:px-6 lg:px-12">
        <div className="mb-4">
          <h1 className="text-3xl font-semibold">Talks</h1>
          <p className="mt-2 text-sm text-default-500">
            Talks and demos from the Sliver ecosystem.
          </p>
        </div>
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-9">
          {talks.map((talk) => (
            <div key={talk.url} className="sm:col-span-1 lg:col-span-3">
              <Card isFooterBlurred className="relative z-0 overflow-hidden">
                <CardHeader className="absolute z-10 top-1 flex-col items-end">
                  <p className="text-xs text-white/70 uppercase font-bold">Talk</p>
                  <p className="text-sm text-right text-white">{talk.title}</p>
                </CardHeader>

                <Youtube
                  className="w-full"
                  url={talk.url}
                  title={talk.title}
                  onPlay={() => hideFooterFor(talk.url)}
                />

                {footerHidden[talk.url] ? null : (
                  <CardFooter className="absolute bottom-0 z-10 bg-black/40 border-t-1 border-default-600 dark:border-default-100">
                    <div className="flex w-full items-center gap-2">
                      <p className="text-xs text-white/80">{talk.description}</p>
                      <Button
                        variant="ghost"
                        color="warning"
                        size="sm"
                        className="ml-auto"
                        onPress={() => {
                          window.open(talk.url, "_blank", "noopener,noreferrer");
                        }}
                      >
                        Watch <FontAwesomeIcon icon={faChevronRight} />
                      </Button>
                    </div>
                  </CardFooter>
                )}
              </Card>
            </div>
          ))}
        </div>
      </div>
    </>
  );
};

export default TalksIndexPage;
