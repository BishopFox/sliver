export type YoutubeProps = {
  className?: string;

  width?: number;
  height?: number;

  embedId: string;
};

export default function Youtube(props: YoutubeProps) {
  return (
    <div className={props.className}>
      <iframe
        width={`${props.width ? props.width : 640}`}
        height={`${props.height ? props.height : 360}`}
        src={`https://youtu.be/embed/${props.embedId}`}
        allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
        allowFullScreen
      />
    </div>
  );
}
