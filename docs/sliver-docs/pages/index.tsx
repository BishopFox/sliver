import AsciinemaPlayer from "@/components/asciinema";
import { SliversIcon } from "@/components/icons/slivers";
import TutorialCard from "@/components/tutorial-card";
import { Themes } from "@/util/themes";
import { Card, CardBody, CardHeader, Divider } from "@nextui-org/react";
import { useRouter } from "next/router";

export default function Home() {
  const router = useRouter();

  function getThemeState(): Themes {
    if (typeof window !== "undefined") {
      const loadedTheme = localStorage.getItem("theme");
      const currentTheme = loadedTheme ? (loadedTheme as Themes) : Themes.DARK;
      return currentTheme;
    }
    return Themes.DARK;
  }

  return (
    <div className="grid grid-cols-12 mt-2">
      <div className="col-span-1"></div>
      <div className="col-span-5 mr-2">
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
      <div className="col-span-5 ml-2">
        <Card>
          <CardHeader>
            <div className="flex items-center">
              <SliversIcon className="mr-2" height={28} />
              <span className="text-2xl">Sliver Command &amp; Control</span>
            </div>
          </CardHeader>
          <Divider />
          <CardBody>
            <p
              className={
                getThemeState() === Themes.DARK
                  ? "prose dark:prose-invert"
                  : "prose prose-slate"
              }
            >
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
          </CardBody>
        </Card>
      </div>
      <div className="col-span-1"></div>

      <div className="col-span-1"></div>
      <div className="col-span-10 mt-8">
        <Divider />
      </div>
      <div className="col-span-1"></div>

      <div className="col-span-1"></div>
      <div className="col-span-10 mt-8">
        <div className="grid grid-cols-8">
          <div className="col-span-2">
            <TutorialCard
              name="Getting Started"
              description="A quick start guide to get you up and running with Sliver."
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
        </div>
      </div>
      <div className="col-span-1"></div>

      <div className="col-span-12 mb-8"></div>
    </div>
  );
}
