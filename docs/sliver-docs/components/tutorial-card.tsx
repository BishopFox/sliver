import { Button, Card } from "@heroui/react";
import AsciinemaPlayer from "./asciinema";

import { faChevronRight } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import styles from "./TutorialCard.module.css";

export type TutorialCardCardProps = {
  name: string;
  description: string;

  asciiCast: string;
  rows?: string;
  cols?: string;
  idleTimeLimit?: number;

  italicDescription?: boolean;
  onPress?: () => void;
  showButton?: boolean;
  buttonText?: string;
};

export default function TutorialCard(props: TutorialCardCardProps) {
  const actionButtonClassName =
    "ml-auto h-auto min-h-0 shrink-0 px-0 py-0 text-sm font-semibold [--button-bg-hover:transparent] [--button-bg-pressed:transparent] [--button-bg:transparent] [--button-fg:var(--warning)]";

  return (
    <Card className="relative z-0 gap-0 overflow-hidden p-0">
      <Card.Header className="absolute left-3 right-3 top-3 z-10 flex-col items-end p-0 text-right">
        <p className="text-sm font-bold uppercase leading-none text-white/70">
          {props.name}
        </p>
      </Card.Header>

      <div className={`${styles["hide-control-bar"]} overflow-hidden`}>
        <AsciinemaPlayer
          src={props.asciiCast}
          rows={props.rows ? props.rows : "18"}
          cols={props.cols ? props.cols : "75"}
          idleTimeLimit={props.idleTimeLimit ? props.idleTimeLimit : 2}
          preload={true}
          autoPlay={true}
          loop={true}
        />
      </div>

      <Card.Footer className="absolute bottom-0 left-0 right-0 z-10 border-t border-white/20 bg-black/40 p-3 backdrop-blur-md">
        <div className="flex min-h-10 w-full flex-row items-center gap-4">
          <p
            className={
              props.italicDescription
                ? "min-w-0 flex-1 text-sm leading-snug text-white/80 italic"
                : "min-w-0 flex-1 text-sm leading-snug text-white/80"
            }
          >
            {props.description}
          </p>

          {props.showButton ? (
            <Button
              variant="ghost"
              size="sm"
              onPress={props.onPress}
              className={actionButtonClassName}
            >
              {props.buttonText ? props.buttonText : "Read Tutorial"}
              <FontAwesomeIcon icon={faChevronRight} />
            </Button>
          ) : (
            <></>
          )}
        </div>
      </Card.Footer>
    </Card>
  );
}
