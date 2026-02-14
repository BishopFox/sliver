import { NextPage } from "next";
import Head from "next/head";
import Youtube from "@/components/youtube";
import { Card, CardBody, CardHeader, Link } from "@heroui/react";
import { title } from "process";

const talks = [
  {
    title: "Building Traffic Encoders",
    url: "https://www.youtube.com/watch?v=6unwFhurm-E",
  },
  {
    title: "Sliver Staging and Automation",
    url: "https://www.youtube.com/watch?v=vuQ5tG5kelI&feature=youtu.be",
  },
  {
    title: "Offensive WASM",
    url: "https://www.youtube.com/watch?v=RnSLsnP4imQ",
  }
];

const TalksIndexPage: NextPage = () => {
  return (
    <>
      <div className="px-4 pb-8 lg:px-8 mt-8">
        <div className="mt-2">
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
            {talks.map((talk) => (
              <Card key={talk.url} className="h-full">
                <CardHeader className="pb-2">
                  <span className="text-base font-semibold">{talk.title}</span>
                </CardHeader>
                <CardBody className="pt-0">
                  <Youtube url={talk.url} title={talk.title} />
                  <Link
                    className="mt-3 inline-flex"
                    href={talk.url}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    Watch on YouTube
                  </Link>
                </CardBody>
              </Card>
            ))}
          </div>
        </div>
      </div>
    </>
  );
};

export default TalksIndexPage;
