import { Card, CardBody } from "@nextui-org/react";

export default function Home() {
  return (
    <div className="grid grid-cols-12 mt-2">
      <div className="col-span-1"></div>
      <div className="col-span-10">
        <Card>
          <CardBody>
            <p>
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
    </div>
  );
}
