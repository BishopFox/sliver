import CodeViewer, { CodeSchema } from "@/components/code";
import { Themes } from "@/util/themes";
import { useTheme } from "next-themes";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/router";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import AsciinemaPlayer from "./asciinema";
import Youtube from "./youtube";

export type MarkdownProps = {
  key?: string;
  markdown: string;
};

type MarkdownAsciiCast = {
  src?: string;
  rows?: string;
  cols?: string;
  idleTimeLimit?: number;
};

const MarkdownViewer = (props: MarkdownProps) => {
  const { theme } = useTheme();
  const router = useRouter();

  return (
    <div
      className={
        theme === Themes.DARK ? "prose dark:prose-invert" : "prose prose-slate"
      }
    >
      <Markdown
        key={props.key || `${Math.random()}`}
        remarkPlugins={[remarkGfm]}
        components={{
          a(props) {
            const { href, children, ...rest } = props;
            if (href?.startsWith("/")) {
              return (
                // @ts-ignore
                <a
                  {...rest}
                  href={href}
                  className="text-primary hover:text-primary-dark"
                  onClick={(e) => {
                    e.preventDefault();
                    router.push(href);
                  }}
                >
                  {children}
                </a>
              );
            }
            const url = new URL(href || "");
            if (url.protocol !== "http:" && url.protocol !== "https:") {
              return <></>;
            }
            if (url.host === "sliver.sh") {
              return (
                // @ts-ignore
                <Link
                  {...rest}
                  href={url.toString()}
                  className="text-primary hover:text-primary-dark"
                >
                  {children}
                </Link>
              );
            }
            return (
              <a
                {...rest}
                href={url.toString()}
                rel="noopener noreferrer"
                target="_blank"
                className="text-primary hover:text-primary-dark"
              >
                {children}
              </a>
            );
          },

          pre(props) {
            // We need to look at the child nodes to avoid wrapping
            // a monaco code block in a <pre> tag
            const { children, className, node, ...rest } = props;
            const childClass = (children as any)?.props?.className;
            if (
              childClass &&
              childClass.startsWith("language-") &&
              childClass !== "language-plaintext"
            ) {
              // @ts-ignore
              return <div {...rest}>{children}</div>;
            }

            return (
              <pre {...rest} className={className}>
                {children}
              </pre>
            );
          },

          img(props) {
            const { src, alt, ...rest } = props;
            return (
              // @ts-ignore
              <Image
                {...rest}
                src={src || ""}
                alt={alt || ""}
                width={500}
                height={500}
                className="w-full rounded-medium"
              />
            );
          },

          code(props) {
            const { children, className, node, ...rest } = props;
            const langTag = /language-(\w+)/.exec(className || "");
            const lang = langTag ? langTag[1] : "plaintext";
            if (lang === "plaintext") {
              return (
                <span className="prose-sm">
                  <code {...rest} className={className}>
                    {children}
                  </code>
                </span>
              );
            }

            if (lang === "youtube") {
              const embedId = (children as string) || "";
              return <Youtube embedId={embedId.trim()} />;
            }

            if (lang === "asciinema") {
              const asciiCast: MarkdownAsciiCast = JSON.parse(
                children as string
              );
              const src = asciiCast.src?.startsWith("/")
                ? `${window.location.origin}${asciiCast.src}`
                : asciiCast.src || "";
              const srcUrl = new URL(src);
              if (srcUrl.protocol !== "http:" && srcUrl.protocol !== "https:") {
                return <></>;
              }
              return (
                <AsciinemaPlayer
                  src={srcUrl.toString()}
                  rows={asciiCast.rows || "18"}
                  cols={asciiCast.cols || "75"}
                  idleTimeLimit={asciiCast.idleTimeLimit || 2}
                  preload={true}
                  autoPlay={true}
                  loop={true}
                />
              );
            }

            const sourceCode = (children as string) || "";
            const lines = sourceCode.split("\n").length;
            return (
              <CodeViewer
                className={
                  lines < 7
                    ? "min-h-[100px]"
                    : lines < 17
                    ? "min-h-[250px]"
                    : "min-h-[450px]"
                }
                key={`${Math.random()}`}
                fontSize={11}
                script={
                  {
                    script_type: lang,
                    source_code: sourceCode,
                  } as CodeSchema
                }
              />
            );
          },
        }}
      >
        {props.markdown}
      </Markdown>
    </div>
  );
};

export default MarkdownViewer;
