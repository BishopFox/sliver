import CodeViewer, { CodeSchema } from "@/components/code";
import { Themes } from "@/util/themes";
import Image from "next/image";
import Link from "next/link";
import { useRouter } from "next/router";
import {
  ComponentPropsWithoutRef,
  ReactNode,
  createElement,
  isValidElement,
  useCallback,
  useEffect,
  useMemo,
  useRef,
} from "react";
import Markdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { useTheme } from "next-themes";
import { Prism as SyntaxHighlighter } from "react-syntax-highlighter";
import {
  oneDark,
  oneLight,
} from "react-syntax-highlighter/dist/cjs/styles/prism";
import AsciinemaPlayer from "./asciinema";
import Mermaid from "./mermaid";
import Youtube from "./youtube";

export type MarkdownProps = {
  markdown: string;
  demoteTopLevelHeading?: boolean;
};

type MarkdownAsciiCast = {
  src?: string;
  rows?: string;
  cols?: string;
  idleTimeLimit?: number;
};

type HeadingLevel = 1 | 2 | 3 | 4 | 5 | 6;

type HeadingProps = ComponentPropsWithoutRef<"h1"> & {
  node?: unknown;
};

const parseMermaidMinHeight = (lang: string, node: any): number | undefined => {
  const [, ...langOptionParts] = lang.split(":");
  const langOptions = langOptionParts.join(":").trim();

  const nodeMeta =
    typeof node?.data?.meta === "string"
      ? node.data.meta
      : typeof node?.meta === "string"
      ? node.meta
      : typeof node?.properties?.metastring === "string"
      ? node.properties.metastring
      : "";

  const optionSources = [langOptions, nodeMeta]
    .map((value) => value.trim())
    .filter(Boolean);

  const parseValue = (raw: string): number | undefined => {
    const directNumber = raw.match(/^(\d{2,4})$/);
    if (directNumber) {
      return Number.parseInt(directNumber[1], 10);
    }

    const keyedNumber = raw.match(
      /(?:^|[,\s;])(?:min[-_]?h(?:eight)?|h(?:eight)?)\s*[:=]\s*(\d{2,4})\b/i
    );
    if (keyedNumber) {
      return Number.parseInt(keyedNumber[1], 10);
    }

    return undefined;
  };

  for (const source of optionSources) {
    const parsed = parseValue(source);
    if (parsed !== undefined && Number.isFinite(parsed)) {
      return Math.min(1200, Math.max(120, parsed));
    }
  }

  return undefined;
};

const mergeClassNames = (
  ...classes: Array<string | false | null | undefined>
) => {
  return classes.filter(Boolean).join(" ");
};

const extractText = (node: ReactNode): string => {
  if (node === null || node === undefined) {
    return "";
  }
  if (typeof node === "string" || typeof node === "number") {
    return String(node);
  }
  if (Array.isArray(node)) {
    return node.map(extractText).join("");
  }
  if (isValidElement(node)) {
    const { children } = node.props as { children?: ReactNode };
    return extractText(children);
  }
  return "";
};

const slugify = (value: string) => {
  return value
    .toLowerCase()
    .trim()
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/[^a-z0-9\s-]/g, "")
    .replace(/\s+/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-|-$/g, "");
};

const headingClassNames: Record<HeadingLevel, string> = {
  1: "mt-12 text-3xl font-semibold tracking-tight text-foreground first:mt-0",
  2: "mt-12 text-2xl font-semibold tracking-tight text-foreground",
  3: "mt-9 text-xl font-semibold tracking-tight text-foreground",
  4: "mt-8 text-lg font-semibold tracking-tight text-foreground",
  5: "mt-7 text-base font-semibold tracking-tight text-foreground",
  6: "mt-7 text-base font-semibold tracking-tight text-foreground",
};

const MarkdownViewer = (props: MarkdownProps) => {
  const { theme } = useTheme();
  const router = useRouter();

  const slugCounterRef = useRef<Map<string, number>>(new Map());

  useEffect(() => {
    slugCounterRef.current = new Map();
  }, [props.markdown]);

  useEffect(() => {
    if (typeof window === "undefined" || !props.markdown) {
      return;
    }

    const hash = router.asPath.split("#")[1];

    if (!hash) {
      return;
    }

    const scrollToHash = () => {
      const target = document.getElementById(hash);
      if (target) {
        target.scrollIntoView({ behavior: "auto", block: "start" });
        return true;
      }
      return false;
    };

    if (scrollToHash()) {
      return;
    }

    let cancelled = false;
    let frameId: number | null = null;
    const maxAttempts = 10;
    let attempts = 0;

    const tryScroll = () => {
      if (cancelled) {
        return;
      }
      if (scrollToHash()) {
        return;
      }
      if (attempts >= maxAttempts) {
        return;
      }
      attempts += 1;
      frameId = window.requestAnimationFrame(tryScroll);
    };

    const timeoutId = window.setTimeout(() => {
      tryScroll();
    }, 0);

    return () => {
      cancelled = true;
      window.clearTimeout(timeoutId);
      if (frameId !== null) {
        window.cancelAnimationFrame(frameId);
      }
    };
  }, [props.markdown, router.asPath]);

  const getAnchor = useCallback((rawValue: string) => {
    const baseSlug = slugify(rawValue);
    const safeBase = baseSlug || `section-${slugCounterRef.current.size + 1}`;
    const usage = slugCounterRef.current.get(safeBase) ?? 0;
    slugCounterRef.current.set(safeBase, usage + 1);
    return usage === 0 ? safeBase : `${safeBase}-${usage}`;
  }, []);

  const headingComponents = useMemo(() => {
    const createHeadingComponent = (level: HeadingLevel) => {
      const HeadingComponent = ({
        children,
        className,
        ...rest
      }: HeadingProps) => {
        const textContent = extractText(children);
        const anchorRef = useRef<string | null>(null);

        if (!anchorRef.current) {
          anchorRef.current = getAnchor(textContent);
        }

        const anchor = anchorRef.current;
        const renderedLevel =
          level === 1 && props.demoteTopLevelHeading ? 2 : level;
        const HeadingTag = `h${renderedLevel}`;

        return createElement(
          HeadingTag,
          {
            ...rest,
            id: anchor || undefined,
            className: mergeClassNames(
              headingClassNames[renderedLevel],
              "scroll-mt-[70px]",
              className
            ),
          },
          <span className="group inline-flex items-baseline gap-2">
            <span className="font-inherit">{children}</span>
            {anchor && (
              <a
                href={`#${anchor}`}
                aria-label={`Link to ${textContent}`}
                className="-my-2 inline-flex size-10 items-center justify-center rounded-xl text-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-accent/40 hover:bg-surface-secondary hover:text-accent"
              >
                <svg
                  aria-hidden="true"
                  className="h-3.5 w-3.5"
                  viewBox="0 0 16 16"
                  fill="none"
                  xmlns="http://www.w3.org/2000/svg"
                >
                  <path
                    d="M7.25 2.75a2.5 2.5 0 0 1 4.243-1.768l1.525 1.525a2.5 2.5 0 0 1 0 3.536l-1.232 1.232"
                    stroke="currentColor"
                    strokeWidth="1.5"
                    strokeLinecap="round"
                  />
                  <path
                    d="M8.75 13.25a2.5 2.5 0 0 1-4.243 1.768l-1.525-1.525a2.5 2.5 0 0 1 0-3.536l1.232-1.232"
                    stroke="currentColor"
                    strokeWidth="1.5"
                    strokeLinecap="round"
                  />
                  <path
                    d="M6 10l4-4"
                    stroke="currentColor"
                    strokeWidth="1.5"
                    strokeLinecap="round"
                  />
                </svg>
              </a>
            )}
          </span>
        );
      };

      HeadingComponent.displayName = `MarkdownHeading${level}`;
      return HeadingComponent;
    };

    return {
      h1: createHeadingComponent(1),
      h2: createHeadingComponent(2),
      h3: createHeadingComponent(3),
      h4: createHeadingComponent(4),
      h5: createHeadingComponent(5),
      h6: createHeadingComponent(6),
    };
  }, [getAnchor, props.demoteTopLevelHeading]);

  const proseClassName = mergeClassNames(
    "markdown-body prose max-w-none leading-7",
    theme === Themes.DARK ? "dark:prose-invert prose-slate" : "prose-slate",
    "prose-code:font-mono prose-img:rounded-2xl"
  );

  return (
    <div className="relative">
      <div className={proseClassName}>
        <Markdown
          remarkPlugins={[remarkGfm]}
          components={{
          ...headingComponents,
          a(anchorProps) {
            const { href, children, className, ...rest } = anchorProps;

            if (href?.startsWith("/")) {
              return (
                <a
                  {...rest}
                  href={href}
                  className={mergeClassNames(
                    "font-medium text-accent underline-offset-4 hover:underline",
                    className
                  )}
                  onClick={(e) => {
                    e.preventDefault();
                    router.push(href);
                  }}
                >
                  {children}
                </a>
              );
            }

            if (!href) {
              return <>{children}</>;
            }

            let url: URL | null = null;
            try {
              const base =
                typeof window !== "undefined"
                  ? window.location.origin
                  : "https://sliver.sh";
              url = /^https?:/i.test(href)
                ? new URL(href)
                : new URL(href, base);
            } catch (error) {
              return <>{children}</>;
            }

            if (url.protocol !== "http:" && url.protocol !== "https:") {
              return <>{children}</>;
            }

            const anchorClassName = mergeClassNames(
              "font-medium text-accent underline-offset-4 hover:underline",
              className
            );

            if (url.host === "sliver.sh") {
              return (
                <Link {...rest} href={url.toString()} className={anchorClassName}>
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
                className={mergeClassNames(
                  anchorClassName,
                  "after:ml-1 after:text-xs after:content-[\"\\2197\"]"
                )}
              >
                {children}
              </a>
            );
          },

          p(paragraphProps) {
            const { className, children, ...rest } = paragraphProps;
            return (
              <p
                {...rest}
                className={mergeClassNames(
                  "my-6 text-base leading-7 text-foreground/80",
                  className
                )}
              >
                {children}
              </p>
            );
          },

          ul(listProps) {
            const { className, children, ...rest } = listProps;
            return (
              <ul
                {...rest}
                className={mergeClassNames(
                  "my-6 list-disc space-y-2 pl-6 text-foreground/80 marker:text-accent",
                  className
                )}
              >
                {children}
              </ul>
            );
          },

          ol(listProps) {
            const { className, children, ...rest } = listProps;
            return (
              <ol
                {...rest}
                className={mergeClassNames(
                  "my-6 list-decimal space-y-2 pl-6 text-foreground/80 marker:text-accent",
                  className
                )}
              >
                {children}
              </ol>
            );
          },

          li(listItemProps) {
            const { className, children, ...rest } = listItemProps;
            return (
              <li
                {...rest}
                className={mergeClassNames(
                  "leading-6 text-foreground/80",
                  className
                )}
              >
                {children}
              </li>
            );
          },

          blockquote(blockquoteProps) {
            const { className, children, ...rest } = blockquoteProps;
            return (
              <blockquote
                {...rest}
                className={mergeClassNames(
                  "my-8 rounded-r-2xl border-l-4 border-accent/40 bg-surface-secondary px-6 py-4 text-base italic text-foreground/80",
                  className
                )}
              >
                {children}
              </blockquote>
            );
          },

          hr(hrProps) {
            const { className, ...rest } = hrProps;
            return (
              <hr
                {...rest}
                className={mergeClassNames(
                  "my-12 border-t border-separator",
                  className
                )}
              />
            );
          },

          table(tableProps) {
            const { className, children, ...rest } = tableProps;
            return (
              <div className="sliver-scrollbar my-8 overflow-x-auto rounded-2xl border border-separator bg-surface">
                <table
                  {...rest}
                  className={mergeClassNames(
                    "w-full min-w-max divide-y divide-separator text-left text-sm",
                    className
                  )}
                >
                  {children}
                </table>
              </div>
            );
          },

          thead(theadProps) {
            const { className, children, ...rest } = theadProps;
            return (
              <thead
                {...rest}
                className={mergeClassNames(
                  "bg-surface-secondary text-sm font-semibold text-muted",
                  className
                )}
              >
                {children}
              </thead>
            );
          },

          tbody(tbodyProps) {
            const { className, children, ...rest } = tbodyProps;
            return (
              <tbody
                {...rest}
                className={mergeClassNames(
                  "divide-y divide-separator",
                  className
                )}
              >
                {children}
              </tbody>
            );
          },

          tr(trProps) {
            const { className, children, ...rest } = trProps;
            return (
              <tr
                {...rest}
                className={mergeClassNames(
                  "bg-transparent",
                  className
                )}
              >
                {children}
              </tr>
            );
          },

          th(thProps) {
            const { className, children, ...rest } = thProps;
            return (
              <th
                {...rest}
                className={mergeClassNames(
                  "px-4 py-3 text-left text-sm font-semibold text-foreground",
                  className
                )}
              >
                {children}
              </th>
            );
          },

          td(tdProps) {
            const { className, children, ...rest } = tdProps;
            return (
              <td
                {...rest}
                className={mergeClassNames(
                  "px-4 py-3 align-top text-sm text-foreground/80",
                  className
                )}
              >
                {children}
              </td>
            );
          },

          strong(strongProps) {
            const { className, children, ...rest } = strongProps;
            return (
              <strong
                {...rest}
                className={mergeClassNames(
                  "font-semibold text-foreground",
                  className
                )}
              >
                {children}
              </strong>
            );
          },

          em(emProps) {
            const { className, children, ...rest } = emProps;
            return (
              <em
                {...rest}
                className={mergeClassNames(
                  "text-foreground/80",
                  className
                )}
              >
                {children}
              </em>
            );
          },

          pre(preProps) {
            // We need to look at the child nodes to avoid wrapping
            // a monaco code block in a <pre> tag
            const { children, className, node, ...rest } = preProps as any;
            const childClass = (children as any)?.props?.className;
            if (
              typeof childClass === "string" &&
              childClass.startsWith("language-mermaid")
            ) {
              return <>{children}</>;
            }
            if (
              childClass &&
              childClass.startsWith("language-") &&
              childClass !== "language-plaintext"
            ) {
              return <div {...rest}>{children}</div>;
            }

            const textContent = extractText(children);

            return (
              <pre
                {...rest}
                className={mergeClassNames(
                  "sliver-scrollbar my-6 overflow-x-auto rounded-2xl border border-separator bg-surface-secondary p-4 text-[13px] leading-6 text-foreground",
                  className
                )}
              >
                {textContent || children}
              </pre>
            );
          },

          img(imageProps) {
            const { src, alt, className, ...rest } = imageProps;
            const imageSrc = typeof src === "string" ? src : "";
            return (
              <Image
                {...rest}
                src={imageSrc}
                alt={alt || ""}
                width={1200}
                height={720}
                className={mergeClassNames(
                  "my-8 w-full rounded-2xl border border-separator/70 object-contain",
                  className
                )}
              />
            );
          },

          code(codeProps) {
            const { inline, children, className, node, ...rest } =
              codeProps as any;

            const languageClass =
              typeof className === "string"
                ? className
                    .split(" ")
                    .find((cls: string) => cls.startsWith("language-"))
                : undefined;

            const lang = languageClass
              ? languageClass.replace("language-", "")
              : "plaintext";
            const [baseLang] = lang.split(":");
            const normalizedLang = baseLang.toLowerCase();
            const childValue = Array.isArray(children)
              ? children.join("")
              : children;
            const sourceCode = typeof childValue === "string" ? childValue : "";

            if (normalizedLang === "youtube") {
              const embedId = sourceCode || "";
              return (
                <div className="not-prose my-8 overflow-hidden rounded-2xl">
                  <Youtube embedId={embedId.trim()} />
                </div>
              );
            }

            if (normalizedLang === "asciinema") {
              const asciiCast: MarkdownAsciiCast = JSON.parse(sourceCode);
              const src = asciiCast.src?.startsWith("/")
                ? `${window.location.origin}${asciiCast.src}`
                : asciiCast.src || "";
              const srcUrl = new URL(src);
              if (srcUrl.protocol !== "http:" && srcUrl.protocol !== "https:") {
                return <></>;
              }
              return (
                <div className="sliver-terminal-frame not-prose my-8 w-full max-w-full overflow-x-auto rounded-2xl bg-[#111315]">
                  <AsciinemaPlayer
                    className="min-w-[42rem]"
                    src={srcUrl.toString()}
                    rows={asciiCast.rows || "18"}
                    cols={asciiCast.cols || "75"}
                    idleTimeLimit={asciiCast.idleTimeLimit || 2}
                    preload={true}
                    autoPlay={true}
                    loop={true}
                  />
                </div>
              );
            }

            if (normalizedLang === "mermaid") {
              return (
                <Mermaid
                  diagram={sourceCode.replace(/\n$/, "")}
                  minHeight={parseMermaidMinHeight(lang, node)}
                />
              );
            }

            if (inline || normalizedLang === "plaintext") {
              return (
                <code
                  {...rest}
                  className={mergeClassNames(
                    "rounded-md bg-surface-secondary px-1.5 py-0.5 font-mono text-[13px] text-foreground",
                    className
                  )}
                >
                  {children}
                </code>
              );
            }

            const baseTheme = theme === Themes.DARK ? oneDark : oneLight;
            const themeOverrides = baseTheme as Record<string, Record<string, unknown>>;
            const preStyles = themeOverrides['pre[class*="language-"]'] || {};
            const codeStyles = themeOverrides['code[class*="language-"]'] || {};
            const syntaxTheme = {
              ...baseTheme,
              'pre[class*="language-"]': {
                ...preStyles,
                background: "transparent",
                backgroundColor: "transparent",
              },
              'code[class*="language-"]': {
                ...codeStyles,
                background: "transparent",
                backgroundColor: "transparent",
              },
            };

            if (normalizedLang.startsWith("monaco")) {
              const rawScriptType = lang.includes(":")
                ? lang.substring(lang.indexOf(":") + 1)
                : lang === "monaco"
                ? "plaintext"
                : lang;
              const scriptType = (rawScriptType || "plaintext").trim() || "plaintext";
              const lines = sourceCode.split("\n").length;
              return (
                <CodeViewer
                  className={
                    lines < 7
                      ? "min-h-[120px]"
                      : lines < 17
                      ? "min-h-[260px]"
                      : "min-h-[480px]"
                  }
                  fontSize={13}
                  script={
                    {
                      script_type: scriptType,
                      source_code: sourceCode,
                    } as CodeSchema
                  }
                />
              );
            }

            const formattedSourceCode = sourceCode.replace(/\n$/, "");

            const preWrapperClassName = mergeClassNames(
              "sliver-scrollbar not-prose mt-4 overflow-x-auto rounded-2xl border border-separator bg-surface-secondary px-4 py-4 text-[13px] leading-6 text-foreground",
              className
            );

            return (
              <pre className={preWrapperClassName}>
                <SyntaxHighlighter
                  language={lang}
                  style={syntaxTheme}
                  PreTag="code"
                  customStyle={{
                    background: "transparent",
                    color: "inherit",
                    margin: 0,
                    padding: 0,
                    fontFamily:
                      "Fira Code, ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, \"Liberation Mono\", \"Courier New\", monospace",
                    fontSize: "inherit",
                    lineHeight: "inherit",
                  }}
                  wrapLongLines={false}
                >
                  {formattedSourceCode}
                </SyntaxHighlighter>
              </pre>
            );
          },
        }}
        >
          {props.markdown}
        </Markdown>
      </div>
    </div>
  );
};

export default MarkdownViewer;
