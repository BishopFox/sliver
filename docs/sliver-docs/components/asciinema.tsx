import "asciinema-player/dist/bundle/asciinema-player.css";
import React from "react";

type AsciinemaPlayerProps = {
  src: string;

  cols?: string;
  rows?: string;
  autoPlay?: boolean;
  preload?: boolean;
  loop?: boolean | number;
  startAt?: number | string;
  speed?: number;
  idleTimeLimit?: number;
  theme?: string;
  poster?: string;
  fit?: string;
  fontSize?: string;
};

function AsciinemaPlayer({ src, ...asciinemaOptions }: AsciinemaPlayerProps) {
  const ref = React.useRef<HTMLDivElement>(null);
  const [player, setPlayer] =
    React.useState<typeof import("asciinema-player")>();
  React.useEffect(() => {
    import("asciinema-player").then((p) => {
      setPlayer(p);
    });
  }, []);
  React.useEffect(() => {
    const currentRef = ref.current;
    const instance = player?.create(src, currentRef, asciinemaOptions);
    return () => {
      instance?.dispose();
    };
  }, [src, player, asciinemaOptions]);

  return <div ref={ref} />;
}

export default AsciinemaPlayer;
