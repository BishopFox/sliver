import Youtube from "@/components/youtube";
import { faArrowUpRightFromSquare } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { Accordion, Card } from "@heroui/react";
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
    title: "Getting Started with Sliver v1.6",
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
  },
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
  const [overlayHidden, setOverlayHidden] = React.useState<
    Record<string, boolean>
  >({});

  const hideOverlayFor = React.useCallback((url: string) => {
    setOverlayHidden((current) =>
      current[url] ? current : { ...current, [url]: true },
    );
  }, []);

  return (
    <>
      <Head>
        <title>Sliver Talks</title>
        <meta
          name="description"
          content="Sliver workshops, technical talks, and community videos."
        />
      </Head>

      <div className="relative overflow-hidden">
        <div className="sliver-grid-surface pointer-events-none absolute inset-x-0 top-0 h-72 opacity-35" />

        <div className="relative mx-auto w-full max-w-7xl px-4 pb-20 pt-8 sm:px-6 sm:pt-10 lg:px-8 lg:pt-12">
          <header className="mb-6 max-w-3xl">
            <p className="text-sm font-medium text-accent">Talks &amp; workshops</p>
            <h1 className="mt-1 text-balance text-3xl font-semibold tracking-tight text-foreground sm:text-4xl">
              Watch Sliver in the field.
            </h1>
            <p className="mt-3 max-w-2xl text-base leading-7 text-muted">
              Practical workshops, technical deep dives, and community sessions
              collected in one compact library.
            </p>
          </header>

          <Accordion
            allowsMultipleExpanded
            defaultExpandedKeys={["workshops", "general-tradecraft"]}
            hideSeparator
            className="w-full px-0"
          >
            {talkSections.map((section) => (
              <Accordion.Item key={section.key} id={section.key}>
                <Accordion.Heading>
                  <Accordion.Trigger>
                    <span className="flex flex-col items-start text-left">
                      <span className="flex items-center gap-2 text-lg font-semibold text-foreground">
                        {section.title}
                        <span className="text-xs font-normal tabular-nums text-muted">
                          {section.talks.length}
                        </span>
                      </span>
                      <span className="mt-1 text-sm text-muted">
                        {section.description}
                      </span>
                    </span>
                    <Accordion.Indicator className="ml-auto" />
                  </Accordion.Trigger>
                </Accordion.Heading>

                <Accordion.Panel>
                  <Accordion.Body>
                    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
                      {section.talks.map((talk) => (
                        <Card
                          key={talk.url}
                          className="relative z-0 gap-0 overflow-hidden bg-black p-0 shadow-surface"
                        >
                          <Youtube
                            className="w-full"
                            url={talk.url}
                            title={talk.title}
                            onInteract={() => hideOverlayFor(talk.url)}
                            onPlay={() => hideOverlayFor(talk.url)}
                          />

                          {overlayHidden[talk.url] ? null : (
                            <>
                              <div
                                aria-hidden="true"
                                className="pointer-events-none absolute inset-x-0 bottom-0 h-32 bg-gradient-to-t from-black/85 via-black/35 to-transparent"
                              />
                              <div className="pointer-events-none absolute inset-x-3 bottom-3 z-10">
                                <div className="flex items-end gap-3 rounded-2xl border border-white/15 bg-black/55 p-3 text-white shadow-xl backdrop-blur-xl">
                                  <div className="min-w-0 flex-1">
                                    <h3 className="text-sm font-semibold leading-snug text-white">
                                      {talk.title}
                                    </h3>
                                    <p className="mt-0.5 truncate text-xs text-white/65">
                                      {talk.description}
                                    </p>
                                  </div>
                                  <a
                                    href={talk.url}
                                    target="_blank"
                                    rel="noreferrer"
                                    aria-label={`Watch ${talk.title} on YouTube`}
                                    className="pointer-events-auto inline-flex min-h-9 shrink-0 items-center gap-1.5 rounded-xl px-2.5 text-xs font-semibold text-white no-underline outline-none transition-colors hover:bg-white/10 focus-visible:ring-2 focus-visible:ring-white/70"
                                  >
                                    Watch
                                    <FontAwesomeIcon
                                      className="text-[10px]"
                                      icon={faArrowUpRightFromSquare}
                                    />
                                  </a>
                                </div>
                              </div>
                            </>
                          )}
                        </Card>
                      ))}
                    </div>
                  </Accordion.Body>
                </Accordion.Panel>
              </Accordion.Item>
            ))}
          </Accordion>
        </div>
      </div>
    </>
  );
};

export default TalksIndexPage;
