import "asciinema-player/dist/bundle/asciinema-player.css";
import React from "react";

type AsciinemaPlayerProps = {
  src: string;
  className?: string;
  hideControls?: boolean;

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

function AsciinemaPlayer({
  src,
  className,
  hideControls,
  cols,
  rows,
  autoPlay,
  preload,
  loop,
  startAt,
  speed,
  idleTimeLimit,
  theme,
  poster,
  fit,
  fontSize,
}: AsciinemaPlayerProps) {
  const ref = React.useRef<HTMLDivElement>(null);
  const [prefersReducedMotion, setPrefersReducedMotion] = React.useState(false);
  const [player, setPlayer] =
    React.useState<typeof import("asciinema-player")>();
  const asciinemaOptions = React.useMemo(
    () => ({
      autoPlay: prefersReducedMotion ? false : autoPlay,
      cols,
      fit,
      fontSize,
      idleTimeLimit,
      loop: prefersReducedMotion ? false : loop,
      poster,
      preload,
      rows,
      speed,
      startAt,
      theme,
    }),
    [
      autoPlay,
      cols,
      fit,
      fontSize,
      idleTimeLimit,
      loop,
      poster,
      preload,
      prefersReducedMotion,
      rows,
      speed,
      startAt,
      theme,
    ],
  );
  React.useEffect(() => {
    const mediaQuery = window.matchMedia("(prefers-reduced-motion: reduce)");
    const syncPreference = () => setPrefersReducedMotion(mediaQuery.matches);
    syncPreference();
    mediaQuery.addEventListener("change", syncPreference);
    return () => mediaQuery.removeEventListener("change", syncPreference);
  }, []);
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

  return (
    <div
      ref={ref}
      className={[
        className,
        hideControls ? "sliver-asciinema-controls-hidden" : undefined,
      ]
        .filter(Boolean)
        .join(" ")}
    />
  );
}

export default AsciinemaPlayer;
