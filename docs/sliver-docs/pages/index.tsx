import AsciinemaPlayer from "@/components/asciinema";
import { SliversIcon } from "@/components/icons/slivers";
import TutorialCard from "@/components/tutorial-card";
import {
  faArrowUpRightFromSquare,
  faDownload,
} from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { Button, Card } from "@heroui/react";
import Head from "next/head";
import Link from "next/link";
import { useRouter } from "next/router";

export default function Home() {
  const router = useRouter();

  return (
    <>
      <Head>
        <title>Sliver C2 Documentation</title>
        <meta
          name="description"
          content="Documentation, tutorials, and community resources for the Sliver command and control framework."
        />
      </Head>

      <div className="relative overflow-hidden">
        <div className="sliver-grid-surface pointer-events-none absolute inset-x-0 top-0 h-[34rem] opacity-55" />

        <div className="relative mx-auto w-full max-w-7xl px-4 pb-20 pt-8 sm:px-6 sm:pt-12 lg:px-8 lg:pt-16">
          <section className="grid min-w-0 grid-cols-1 items-stretch gap-6 lg:grid-cols-12">
            <div className="min-w-0 lg:col-span-7">
              <Card className="relative h-full min-w-0 gap-0 overflow-hidden rounded-3xl bg-black p-0 shadow-surface">
                <div className="absolute right-3 top-3 z-10 max-w-[calc(100%_-_1.5rem)]">
                  <Link
                    href="/tutorials"
                    aria-label="Browse Sliver tutorials"
                    className="inline-flex min-w-0 cursor-pointer items-center gap-2.5 rounded-xl border border-white/15 bg-black/55 px-3 py-2 text-white no-underline shadow-lg outline-none backdrop-blur-xl hover:bg-black/70 focus-visible:ring-2 focus-visible:ring-white/70"
                  >
                    <span className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-white/10 text-white/90">
                      <SliversIcon height={16} />
                    </span>
                    <div className="min-w-0">
                      <p className="truncate text-xs font-semibold">
                        Sliver operator console
                      </p>
                      <p className="truncate text-[11px] text-white/60">
                        Interactive command tour
                      </p>
                    </div>
                  </Link>
                </div>

                <div className="sliver-terminal-frame min-h-[20rem] flex-1 overflow-x-auto bg-[#111315] sm:min-h-[24rem]">
                  <AsciinemaPlayer
                    className="sliver-asciinema-fit-height h-full min-w-[35rem]"
                    src="/asciinema/intro.cast"
                    fit="height"
                    hideControls
                    rows="18"
                    cols="75"
                    idleTimeLimit={60}
                    preload={true}
                    autoPlay={true}
                    loop={true}
                  />
                </div>
              </Card>
            </div>

            <Card className="h-full bg-surface/80 shadow-surface backdrop-blur-2xl lg:col-span-5">
              <Card.Header className="pb-3">
                <div className="min-w-0">
                  <p className="text-sm font-medium text-accent">
                    Operator documentation
                  </p>
                  <h1 className="mt-1 text-balance text-3xl font-semibold tracking-tight text-foreground sm:text-4xl">
                    Sliver Command &amp; Control
                  </h1>
                </div>
              </Card.Header>

              <Card.Content className="pt-2">
                <p className="text-pretty text-base leading-7 text-muted">
                  Sliver is a cross-platform command and control framework for
                  professional red teams. Operate over mTLS, WireGuard, HTTP(S),
                  and DNS from one console, with broad operating-system and CPU
                  architecture support.
                </p>
                <p className="mt-4 text-sm leading-6 text-muted">
                  Start with the operator guide, download the latest release, or
                  extend your workflow with community packages from the Armory.
                </p>
              </Card.Content>

              <Card.Footer className="mt-auto flex-col items-stretch gap-3 pt-5 sm:flex-row">
                <Button
                  className="sm:flex-1"
                  variant="primary"
                  onPress={() =>
                    router.push({
                      pathname: "/docs",
                      query: { name: "Getting Started" },
                    })
                  }
                >
                  Get started
                </Button>
                <Button
                  className="sm:flex-1"
                  variant="outline"
                  onPress={() => {
                    window.open(
                      "https://github.com/BishopFox/sliver/releases/latest",
                      "_blank",
                      "noopener,noreferrer",
                    );
                  }}
                >
                  <FontAwesomeIcon icon={faDownload} />
                  Latest release
                </Button>
              </Card.Footer>

              <a
                href="https://github.com/sliverarmory"
                target="_blank"
                rel="noreferrer"
                className="mx-6 mb-6 mt-1 inline-flex w-fit items-center gap-1.5 rounded-lg text-sm font-medium text-accent no-underline outline-none hover:underline focus-visible:ring-2 focus-visible:ring-accent/40"
              >
                Browse the Armory
                <FontAwesomeIcon
                  className="text-xs"
                  icon={faArrowUpRightFromSquare}
                />
              </a>
            </Card>
          </section>

          <section
            aria-labelledby="quick-guides-heading"
            className="mt-12 border-t border-separator/70 pt-10"
          >
            <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
              <div>
                <p className="text-sm font-medium text-accent">Quick guides</p>
                <h2
                  id="quick-guides-heading"
                  className="mt-1 text-2xl font-semibold tracking-tight text-foreground sm:text-3xl"
                >
                  Learn the workflow in the console.
                </h2>
              </div>
              <Button
                variant="ghost"
                onPress={() => router.push("/tutorials")}
              >
                Browse all tutorials
              </Button>
            </div>

            <div className="mt-6 grid min-w-0 grid-cols-1 gap-5 xl:grid-cols-2">
              <TutorialCard
                name="Getting Started"
                description="Install Sliver and open your first operator workflow."
                asciiCast="/asciinema/install-1.cast"
                cols="133"
                rows="32"
                idleTimeLimit={1}
                hideControls
                showButton={true}
                buttonText="Read guide"
                onPress={() => {
                  router.push({
                    pathname: "/docs",
                    query: { name: "Getting Started" },
                  });
                }}
              />

              <TutorialCard
                name="Compile from Source"
                description="Build a development copy and understand the toolchain."
                asciiCast="/asciinema/compile-from-source.cast"
                cols="133"
                rows="32"
                idleTimeLimit={1}
                hideControls
                showButton={true}
                buttonText="Read guide"
                onPress={() => {
                  router.push({
                    pathname: "/docs",
                    query: { name: "Compile from Source" },
                  });
                }}
              />
            </div>
          </section>
        </div>
      </div>
    </>
  );
}
