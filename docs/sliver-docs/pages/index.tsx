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
          idleTimeLimit={20}
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
              Sliver is a Command and Control (C2) system made for penetration
              testers, red teams, and blue teams. It generates implants that can
              run on virtually every architecture out there, and securely manage
              these connections through a central server. Sliver supports
              multiple callback protocols including DNS, Mutual TLS (mTLS),
              WireGuard, and HTTP(S) to make egress simple, even when those
              pesky blue teams block your domains. You can even have multiple
              operators (players) simultaneously commanding your sliver army.
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
                router.push("/docs?name=Getting%20Started");
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
