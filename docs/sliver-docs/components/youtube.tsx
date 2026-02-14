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

export default function Youtube(props: YoutubeProps) {
  const embedId = resolveYoutubeEmbedId(props.embedId, props.url);
  if (!embedId) {
    return null;
  }

  const startAt = Number.isFinite(props.startAt) && props.startAt && props.startAt > 0
    ? `&start=${Math.floor(props.startAt)}`
    : "";
  const src = `https://www.youtube-nocookie.com/embed/${embedId}?rel=0${startAt}`;
  const title = props.title || "YouTube video player";
  const responsive = props.responsive ?? true;

  if (responsive) {
    return (
      <div className={props.className}>
        <div className="relative w-full overflow-hidden rounded-xl bg-black pb-[56.25%]">
          <iframe
            className={`absolute left-0 top-0 h-full w-full ${props.iframeClassName || ""}`.trim()}
            src={src}
            title={title}
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
    <div className={props.className}>
      <iframe
        width={`${props.width ? props.width : 640}`}
        height={`${props.height ? props.height : 360}`}
        className={props.iframeClassName}
        src={src}
        title={title}
        loading="lazy"
        referrerPolicy="strict-origin-when-cross-origin"
        allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
        allowFullScreen
      />
    </div>
  );
}
