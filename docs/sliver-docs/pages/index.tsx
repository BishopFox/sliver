import AsciinemaPlayer from "@/components/asciinema";
import { SliversIcon } from "@/components/icons/slivers";
import TutorialCard from "@/components/tutorial-card";
import { Themes } from "@/util/themes";
import { faDownload, faExternalLink } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { Button, Card, Separator } from "@heroui/react";
import { useTheme } from "next-themes";
import { useRouter } from "next/router";
import React from "react";

export default function Home() {
  const router = useRouter();
  const { resolvedTheme } = useTheme();

  const isDarkTheme = React.useMemo(() => {
    return (resolvedTheme || Themes.DARK) !== Themes.LIGHT;
  }, [resolvedTheme]);

  return (
    <div className="mt-6 flex flex-col gap-6 px-4 sm:px-6 lg:grid lg:grid-cols-12 lg:px-12">
      <div className="lg:col-span-6">
        <div className="w-full overflow-hidden rounded-xl border border-separator bg-surface shadow-sm">
          <div className="w-full overflow-x-auto lg:overflow-visible">
            <AsciinemaPlayer
              src="/asciinema/intro.cast"
              rows="18"
              cols="75"
              idleTimeLimit={60}
              preload={true}
              autoPlay={true}
              loop={true}
            />
          </div>
        </div>
      </div>
      <div className="lg:col-span-6 lg:ml-2">
        <Card className="mx-auto max-w-3xl lg:mx-0">
          <Card.Header>
            <div className="flex items-center">
              <SliversIcon className="mr-2" height={28} />
              <span className="text-2xl">Sliver Command &amp; Control</span>
            </div>
          </Card.Header>
          <Separator />
          <Card.Content>
            <p className={isDarkTheme ? "prose dark:prose-invert" : "prose prose-slate"}>
              Sliver is a powerful command and control (C2) framework designed
              to provide advanced capabilities for covertly managing and
              controlling remote systems. With Sliver, security professionals,
              red teams, and penetration testers can easily establish a secure
              and reliable communication channel over Mutual TLS, HTTP(S), DNS,
              or Wireguard with target machines. Enabling them to execute
              commands, gather information, and perform various
              post-exploitation activities. The framework offers a user-friendly
              console interface, extensive functionality, and support for
              multiple operating systems as well as multiple CPU architectures,
              making it an indispensable tool for conducting comprehensive
              offensive security operations.
            </p>
            <div className="mt-4 flex w-full gap-3">
              <Button
                className="flex-1"
                variant="primary"
                onPress={() => {
                  window.open(
                    "https://github.com/BishopFox/sliver/releases/latest",
                    "_blank",
                    "noopener,noreferrer",
                  );
                }}
              >
                <FontAwesomeIcon icon={faDownload} />
                Download Latest Release
              </Button>
              <Button
                className="flex-1"
                variant="secondary"
                onPress={() => {
                  window.open(
                    "https://github.com/sliverarmory",
                    "_blank",
                    "noopener,noreferrer",
                  );
                }}
              >
                Visit the Armory
                <FontAwesomeIcon icon={faExternalLink} />
              </Button>
            </div>
          </Card.Content>
        </Card>
      </div>

      <div className="col-span-12 mt-8">
        <Separator />
      </div>

      <div className="col-span-12 mt-8">
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-9">
          <div className="sm:col-span-1 lg:col-span-3">
            <TutorialCard
              name="Getting Started"
              description="A quick start guide to get you up and running"
              asciiCast="/asciinema/install-1.cast"
              cols="133"
              rows="32"
              idleTimeLimit={1}
              showButton={true}
              buttonText="Read Docs"
              onPress={() => {
                router.push({
                  pathname: "/docs",
                  query: { name: "Getting Started" },
                });
              }}
            />
          </div>

          <div className="sm:col-span-1 lg:col-span-3">
            <TutorialCard
              name="Compile From Source"
              description="How to compile Sliver from source"
              asciiCast="/asciinema/compile-from-source.cast"
              cols="133"
              rows="32"
              idleTimeLimit={1}
              showButton={true}
              buttonText="Read Docs"
              onPress={() => {
                router.push({
                  pathname: "/docs",
                  query: { name: "Compile from Source" },
                });
              }}
            />
          </div>
        </div>
      </div>

      <div className="col-span-12 mb-8"></div>
    </div>
  );
}
