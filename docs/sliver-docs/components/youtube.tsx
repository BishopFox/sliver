import React from "react";

type YoutubePlayerState = {
  PLAYING: number;
};

type YoutubePlayerStateChangeEvent = {
  data: number;
};

type YoutubePlayer = {
  destroy?: () => void;
};

type YoutubePlayerCtor = new (
  element: HTMLIFrameElement | string,
  options?: {
    events?: {
      onStateChange?: (event: YoutubePlayerStateChangeEvent) => void;
    };
  }
) => YoutubePlayer;

type YoutubeNamespace = {
  Player: YoutubePlayerCtor;
  PlayerState: YoutubePlayerState;
};

declare global {
  interface Window {
    YT?: YoutubeNamespace;
    onYouTubeIframeAPIReady?: () => void;
    __sliverYoutubeApiPromise?: Promise<YoutubeNamespace>;
  }
}

export type YoutubeProps = {
  className?: string;
  iframeClassName?: string;

  width?: number;
  height?: number;
  responsive?: boolean;
  title?: string;
  startAt?: number;

  embedId?: string;
  url?: string;

  onInteract?: () => void;
  onPlay?: () => void;
};

const YOUTUBE_ID_PATTERN = /^[A-Za-z0-9_-]{11}$/;

function parseYoutubeEmbedIdFromUrl(input: string): string | null {
  try {
    const parsed = new URL(input);
    const host = parsed.hostname.toLowerCase().replace(/^www\./, "");
    const parts = parsed.pathname.split("/").filter(Boolean);

    let candidate = "";
    if (host === "youtu.be" && parts.length > 0) {
      candidate = parts[0] || "";
    } else if (host === "youtube.com" || host === "m.youtube.com") {
      if (parts[0] === "watch") {
        candidate = parsed.searchParams.get("v") || "";
      } else if ((parts[0] === "embed" || parts[0] === "shorts") && parts.length > 1) {
        candidate = parts[1] || "";
      }
    }

    return YOUTUBE_ID_PATTERN.test(candidate) ? candidate : null;
  } catch {
    return null;
  }
}

function resolveYoutubeEmbedId(embedId?: string, url?: string): string | null {
  const idCandidate = (embedId || "").trim();
  if (YOUTUBE_ID_PATTERN.test(idCandidate)) {
    return idCandidate;
  }
  if (idCandidate) {
    const parsedFromEmbedId = parseYoutubeEmbedIdFromUrl(idCandidate);
    if (parsedFromEmbedId) {
      return parsedFromEmbedId;
    }
  }

  const urlCandidate = (url || "").trim();
  if (urlCandidate) {
    return parseYoutubeEmbedIdFromUrl(urlCandidate);
  }
  return null;
}

function loadYoutubeIframeApi(): Promise<YoutubeNamespace> {
  if (typeof window === "undefined") {
    return Promise.reject(new Error("YouTube API cannot load during SSR"));
  }
  if (window.YT?.Player) {
    return Promise.resolve(window.YT);
  }
  if (window.__sliverYoutubeApiPromise) {
    return window.__sliverYoutubeApiPromise;
  }

  window.__sliverYoutubeApiPromise = new Promise<YoutubeNamespace>((resolve, reject) => {
    const previousReadyHandler = window.onYouTubeIframeAPIReady;
    window.onYouTubeIframeAPIReady = () => {
      previousReadyHandler?.();
      if (window.YT?.Player) {
        resolve(window.YT);
      } else {
        reject(new Error("YouTube API loaded without Player support"));
      }
    };

    const existingScript = document.querySelector<HTMLScriptElement>(
      "script[data-youtube-iframe-api='true']"
    );
    if (existingScript) {
      return;
    }

    const script = document.createElement("script");
    script.src = "https://www.youtube.com/iframe_api";
    script.async = true;
    script.dataset.youtubeIframeApi = "true";
    script.onerror = () => reject(new Error("Failed to load YouTube iframe API"));
    document.head.appendChild(script);
  });

  return window.__sliverYoutubeApiPromise;
}

export default function Youtube(props: YoutubeProps) {
  const {
    className,
    iframeClassName,
    width,
    height,
    responsive,
    title,
    startAt,
    embedId: embedIdProp,
    url,
    onInteract,
    onPlay,
  } = props;
  const embedId = resolveYoutubeEmbedId(embedIdProp, url);
  const iframeRef = React.useRef<HTMLIFrameElement | null>(null);
  const onPlayRef = React.useRef(onPlay);

  React.useEffect(() => {
    onPlayRef.current = onPlay;
  }, [onPlay]);

  React.useEffect(() => {
    if (!embedId || !onPlayRef.current || !iframeRef.current) {
      return;
    }

    let player: YoutubePlayer | null = null;
    let disposed = false;

    loadYoutubeIframeApi()
      .then((YT) => {
        if (disposed || !iframeRef.current) {
          return;
        }

        player = new YT.Player(iframeRef.current, {
          events: {
            onStateChange: (event) => {
              if (event.data === YT.PlayerState.PLAYING) {
                onPlayRef.current?.();
              }
            },
          },
        });
      })
      .catch(() => {
        // No-op: if API fails, videos still load via standard iframe behavior.
      });

    return () => {
      disposed = true;
      player?.destroy?.();
    };
  }, [embedId]);

  if (!embedId) {
    return null;
  }

  const startAtParam =
    Number.isFinite(startAt) && startAt && startAt > 0
      ? `&start=${Math.floor(startAt)}`
      : "";
  const enableJsApi = onPlay ? "&enablejsapi=1" : "";
  const src = `https://www.youtube-nocookie.com/embed/${embedId}?rel=0${startAtParam}${enableJsApi}`;
  const playerTitle = title || "YouTube video player";
  const isResponsive = responsive ?? true;
  const handleInteract = () => onInteract?.();

  if (isResponsive) {
    return (
      <div className={className} onPointerDown={handleInteract}>
        <div className="relative w-full overflow-hidden rounded-xl bg-black pb-[56.25%]">
          <iframe
            ref={iframeRef}
            className={`absolute left-0 top-0 h-full w-full ${iframeClassName || ""}`.trim()}
            src={src}
            title={playerTitle}
            loading="lazy"
            referrerPolicy="strict-origin-when-cross-origin"
            allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
            allowFullScreen
          />
        </div>
      </div>
    );
  }

  return (
    <div className={className} onPointerDown={handleInteract}>
      <iframe
        ref={iframeRef}
        width={`${width ? width : 640}`}
        height={`${height ? height : 360}`}
        className={iframeClassName}
        src={src}
        title={playerTitle}
        loading="lazy"
        referrerPolicy="strict-origin-when-cross-origin"
        allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
        allowFullScreen
      />
    </div>
  );
}
