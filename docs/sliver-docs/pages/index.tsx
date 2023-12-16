import AsciinemaPlayer from "@/components/asciinema";
import { Themes } from "@/util/themes";
import { Card, CardBody, CardHeader, Divider } from "@nextui-org/react";

export default function Home() {
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
          idleTimeLimit={2}
          preload={true}
          autoPlay={true}
          loop={true}
        />
      </div>
      <div className="col-span-5 ml-2">
        <Card>
          <CardHeader>
            <span className="text-2xl">Sliver Command &amp; Control</span>
          </CardHeader>
          <Divider />
          <CardBody>
            <p
              className={
                getThemeState() === Themes.DARK
                  ? "prose prose-sm dark:prose-invert"
                  : "prose prose-sm prose-slate"
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

      {/* <div className="col-span-1"></div>
      <div className="col-span-10 mt-2">
        <Divider />
      </div>
      <div className="col-span-1"></div> */}
    </div>
  );
}
