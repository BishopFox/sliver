import { Button, Card } from "@heroui/react";
import AsciinemaPlayer from "./asciinema";

import { faChevronRight } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";

export type TutorialCardCardProps = {
  name: string;
  description: string;

  asciiCast: string;
  rows?: string;
  cols?: string;
  idleTimeLimit?: number;
  hideControls?: boolean;

  italicDescription?: boolean;
  onPress?: () => void;
  showButton?: boolean;
  buttonText?: string;
};

export default function TutorialCard(props: TutorialCardCardProps) {
  return (
    <Card className="relative h-full min-w-0 gap-0 overflow-hidden bg-black p-0 shadow-surface">
      <div className="sliver-terminal-frame relative w-full max-w-full overflow-x-auto bg-[#111315]">
        <AsciinemaPlayer
          className="min-w-[36rem]"
          src={props.asciiCast}
          hideControls={props.hideControls}
          rows={props.rows || "18"}
          cols={props.cols || "75"}
          idleTimeLimit={props.idleTimeLimit || 2}
          preload={true}
          autoPlay={true}
          loop={true}
        />

        <div className="pointer-events-none absolute inset-x-3 top-3 z-10 flex justify-end">
          <h3 className="max-w-[80%] rounded-xl border border-white/15 bg-black/55 px-3 py-2 text-right text-sm font-semibold text-white/85 shadow-lg backdrop-blur-xl">
            {props.name}
          </h3>
        </div>
      </div>

      <Card.Footer className="absolute inset-x-3 bottom-10 z-10 flex-row items-center gap-3 rounded-2xl border border-white/15 bg-black/55 px-3 py-2.5 text-white shadow-xl backdrop-blur-xl">
        <p
          className={`min-w-0 flex-1 text-sm leading-5 text-white/70 ${
            props.italicDescription ? "italic" : ""
          }`}
        >
          {props.description}
        </p>

        {props.showButton ? (
          <Button
            className="shrink-0 text-white hover:bg-white/10"
            size="sm"
            variant="ghost"
            onPress={props.onPress}
          >
            {props.buttonText || "Read tutorial"}
            <FontAwesomeIcon icon={faChevronRight} />
          </Button>
        ) : null}
      </Card.Footer>
    </Card>
  );
}
