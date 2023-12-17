import { Button, Card, CardFooter, CardHeader } from "@nextui-org/react";
import AsciinemaPlayer from "./asciinema";

import styles from "./TutorialCard.module.css";

export type TutorialCardCardProps = {
  name: string;
  description: string;

  italicDescription?: boolean;
  onPress?: () => void;
  showButton?: boolean;
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
          src="/asciinema/intro.cast"
          rows="18"
          cols="75"
          idleTimeLimit={20}
          preload={true}
          autoPlay={true}
          loop={true}
        />
      </div>

      <CardFooter className="absolute bg-black/40 bottom-0 z-100 border-t-1 border-default-600 dark:border-default-100">
        <div className="w-full flex items-center">
          <div className="w-full">
            <p
              className={
                props.italicDescription
                  ? "text-xs text-white/70 italic"
                  : "text-xs text-white/70"
              }
            >
              {props.description}
            </p>
          </div>
          {props.showButton ? (
            <div className="w-full grid justify-items-end">
              <Button
                variant="ghost"
                color="warning"
                size="sm"
                onPress={props.onPress}
              >
                &nbsp;&nbsp;Help&nbsp;&nbsp;
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
