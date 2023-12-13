export type SliversIconProps = {
  className?: string;
  fill?: string;

  height?: number;
  width?: number;
};

export const SliversIcon = (props: SliversIconProps) => {
  return (
    <svg
      aria-hidden="true"
      focusable="false"
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 912 1024"
      className={props.className}
      height={props.height || 24}
      width={props.width}
    >
      <path
        fill={props.fill ? props.fill : "currentColor"}
        d="M3.767 709.26s-38.415-249.17 142.452-440.946C326.97 76.538 584.482 63.816 759.689 133.127c0 0-32.847 29.271-32.847 65.806 0 36.444 21.936 58.378 45.658 82.101 23.723 23.815 135.095 158.909 135.095 312.294 0 153.453-125.972 336.015-324.992 336.015-199.042 0-312.293-144.239-328.682-310.414 0 0 96.771 164.294 255.611 164.294s213.62-125.971 213.62-219.167c0-93.034-51.113-272.02-275.666-272.02-224.645 0-379.795 199.042-443.72 417.224z"
      ></path>
    </svg>
  );
};
