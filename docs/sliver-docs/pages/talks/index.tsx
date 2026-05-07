import Youtube from "@/components/youtube";
import { faChevronRight } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { Accordion, Button, Card } from "@heroui/react";
import { NextPage } from "next";
import Head from "next/head";
import React from "react";

type Talk = {
  title: string;
  description: string;
  url: string;
};

type TalkSection = {
  key: "workshops" | "general-tradecraft" | "community";
  title: "Workshops" | "General Tradecraft" | "Community";
  description: string;
  talks: Talk[];
};

const workshopTalks: Talk[] = [
  {
    title:"Getting Started with Sliver v1.6",
    description: "Introductory workshop covering basics and new features.",
    url: "https://www.youtube.com/watch?v=IOiyXYp1lDc",
  },
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
];

const generalTradecraftTalks: Talk[] = [
  {
    title: "Offensive WASM",
    description: "Applying WebAssembly techniques in offensive tradecraft.",
    url: "https://www.youtube.com/watch?v=RnSLsnP4imQ",
  },
  {
    title: "The Sliver C2 Framework",
    description: "General discussion of C2 design.",
    url: "https://www.youtube.com/watch?v=tkjMZuZ_8nw",
  }
];

const communityTalks: Talk[] = [
  {
    title: "Community Guide Video 1",
    description: "Linked from Community Guides.",
    url: "https://youtu.be/3R6WKUgN0K4?t=456",
  },
  {
    title: "Community Guide Video 2",
    description: "Linked from Community Guides.",
    url: "https://www.youtube.com/watch?v=QO_1UMaiWHk",
  },
  {
    title: "Community Guide Video 3",
    description: "Linked from Community Guides.",
    url: "https://www.youtube.com/watch?v=izMMmOaLn9g",
  },
  {
    title: "Community Guide Video 4",
    description: "Linked from Community Guides.",
    url: "https://www.youtube.com/watch?v=qIbrozlf2wM",
  },
  {
    title: "Community Guide Video 5",
    description: "Linked from Community Guides.",
    url: "https://www.youtube.com/watch?v=CKfjLnEMfvI",
  },
];

const talkSections: TalkSection[] = [
  {
    key: "workshops",
    title: "Workshops",
    description: "Hands-on workshop recordings focused on Sliver workflows.",
    talks: workshopTalks,
  },
  {
    key: "general-tradecraft",
    title: "General Tradecraft",
    description: "Broader offensive engineering and tradecraft talks.",
    talks: generalTradecraftTalks,
  },
  {
    key: "community",
    title: "Community",
    description: "Community-created videos listed in Community Guides.",
    talks: communityTalks,
  },
];

const TalksIndexPage: NextPage = () => {
  const [footerHidden, setFooterHidden] = React.useState<Record<string, boolean>>(
    {}
  );
  const actionButtonClassName =
    "ml-auto h-auto min-h-0 shrink-0 px-0 py-0 text-sm font-semibold [--button-bg-hover:transparent] [--button-bg-pressed:transparent] [--button-bg:transparent] [--button-fg:var(--warning)]";

  const hideFooterFor = React.useCallback((url: string) => {
    setFooterHidden((prev) => (prev[url] ? prev : { ...prev, [url]: true }));
  }, []);

  return (
    <>
      <Head>
        <title>Sliver Talks</title>
      </Head>
      <div className="mt-6 px-4 pb-8 sm:px-6 lg:px-12">
        <Accordion
          allowsMultipleExpanded
          defaultExpandedKeys={["workshops", "general-tradecraft"]}
          hideSeparator
          className="px-0"
        >
          {talkSections.map((section) => (
            <Accordion.Item
              key={section.key}
              id={section.key}
            >
              <Accordion.Heading>
                <Accordion.Trigger>
                  <span className="flex flex-col items-start text-left">
                    <span className="text-lg font-semibold">{section.title}</span>
                    <span className="text-sm text-muted">{section.description}</span>
                  </span>
                  <Accordion.Indicator className="ml-auto" />
                </Accordion.Trigger>
              </Accordion.Heading>
              <Accordion.Panel>
                <Accordion.Body>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-9">
                {section.talks.map((talk) => (
                  <div key={talk.url} className="sm:col-span-1 lg:col-span-3">
                    <Card className="relative z-0 gap-0 overflow-hidden p-0">
                      <Youtube
                        className="w-full"
                        url={talk.url}
                        title={talk.title}
                        onPlay={() => hideFooterFor(talk.url)}
                      />

                      {footerHidden[talk.url] ? null : (
                        <Card.Footer className="absolute bottom-0 left-0 right-0 z-10 border-t border-white/20 bg-black/40 p-3 backdrop-blur-md">
                          <div className="flex min-h-10 w-full items-center gap-4">
                            <p className="min-w-0 flex-1 text-sm leading-snug text-white/80">{talk.description}</p>
                            <Button
                              variant="ghost"
                              size="sm"
                              className={actionButtonClassName}
                              onPress={() => {
                                window.open(talk.url, "_blank", "noopener,noreferrer");
                              }}
                            >
                              Watch <FontAwesomeIcon icon={faChevronRight} />
                            </Button>
                          </div>
                        </Card.Footer>
                      )}
                    </Card>
                  </div>
                ))}
              </div>
                </Accordion.Body>
              </Accordion.Panel>
            </Accordion.Item>
          ))}
        </Accordion>
      </div>
    </>
  );
};

export default TalksIndexPage;
