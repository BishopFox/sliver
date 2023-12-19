import { Button, Card, CardFooter, CardHeader } from "@nextui-org/react";
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
  return (
    <Card isFooterBlurred>
      <CardHeader className="absolute z-10 top-1 flex-col items-end">
        <p className="text-md  text-white/70 uppercase font-bold">
          {props.name}
        </p>
      </CardHeader>

      <div className={styles["hide-control-bar"]}>
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

      <CardFooter className="absolute bg-black/40 bottom-0 z-100 border-t-1 border-default-600 dark:border-default-100">
        <div className="flex w-full items-center">
          <p
            className={
              props.italicDescription
                ? "text-xs text-white/80 italic"
                : "text-xs text-white/80"
            }
          >
            {props.description}
          </p>

          {props.showButton ? (
            <div className="justify-items-end">
              <Button
                variant="ghost"
                color="warning"
                size="sm"
                onPress={props.onPress}
              >
                {props.buttonText ? props.buttonText : "Read Tutorial"}{" "}
                <FontAwesomeIcon icon={faChevronRight} />
              </Button>
            </div>
          ) : (
            <></>
          )}
        </div>
      </CardFooter>
    </Card>
  );
}
