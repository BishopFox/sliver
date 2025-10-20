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

type HeadingLevel = 1 | 2 | 3 | 4 | 5 | 6;

type HeadingProps = ComponentPropsWithoutRef<"h1"> & {
  node?: unknown;
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
  1: "mt-12 text-3xl font-semibold tracking-tight text-slate-900 first:mt-0 dark:text-slate-100",
  2: "mt-10 text-2xl font-semibold tracking-tight text-slate-900 dark:text-slate-100",
  3: "mt-8 text-xl font-semibold tracking-tight text-slate-900 dark:text-slate-100",
  4: "mt-6 text-lg font-semibold tracking-tight text-slate-900 dark:text-slate-100",
  5: "mt-6 text-base font-semibold tracking-tight text-slate-900 dark:text-slate-100",
  6: "mt-6 text-base font-semibold uppercase tracking-wide text-slate-700 dark:text-slate-300",
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
        const HeadingTag = `h${level}`;

        return createElement(
          HeadingTag,
          {
            ...rest,
            id: anchor || undefined,
            className: mergeClassNames(
              headingClassNames[level],
              "scroll-mt-[70px]",
              className
            ),
          },
          <span className="group inline-flex items-baseline gap-2">
            <span className="font-inherit">{children}</span>
            {anchor && (
              <a
                href={`#${anchor}`}
                aria-label="Copy link to section"
                className="inline-flex h-6 w-6 items-center justify-center rounded-md text-slate-400 transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40 focus-visible:ring-offset-2 hover:text-primary dark:text-slate-500"
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
  }, [getAnchor]);

  const proseClassName = mergeClassNames(
    "markdown-body prose max-w-none leading-7",
    theme === Themes.DARK ? "dark:prose-invert prose-slate" : "prose-slate",
    "prose-pre:bg-slate-950/90 prose-pre:text-slate-100 prose-code:font-mono prose-img:rounded-xl"
  );

  return (
    <div className="relative">
      <div className={proseClassName}>
        <Markdown
          key={props.key || `${Math.random()}`}
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
                    "font-medium text-primary transition-colors duration-150 hover:text-primary-dark",
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
              "font-medium text-primary transition-colors duration-150 underline-offset-4 hover:text-primary-dark",
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
                  "my-6 text-base leading-7 text-slate-700 dark:text-slate-300",
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
                  "my-6 list-disc space-y-2 pl-6 text-slate-700 marker:text-primary dark:text-slate-300",
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
                  "my-6 list-decimal space-y-2 pl-6 text-slate-700 marker:text-primary dark:text-slate-300",
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
                  "leading-6 text-slate-700 dark:text-slate-300",
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
                  "my-8 border-l-4 border-slate-200 bg-slate-50 px-6 py-4 text-base italic text-slate-700 shadow-sm dark:border-slate-700 dark:bg-slate-900/50 dark:text-slate-300",
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
                  "my-12 border-t border-slate-200 dark:border-slate-800",
                  className
                )}
              />
            );
          },

          table(tableProps) {
            const { className, children, ...rest } = tableProps;
            return (
              <div className="my-8 overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm dark:border-slate-800 dark:bg-slate-950/40">
                <table
                  {...rest}
                  className={mergeClassNames(
                    "w-full min-w-max divide-y divide-slate-200 text-left text-sm dark:divide-slate-800",
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
                  "bg-slate-50 text-sm font-semibold uppercase tracking-wide text-slate-600 dark:bg-slate-900/60 dark:text-slate-300",
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
                  "divide-y divide-slate-200 dark:divide-slate-800",
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
                  "transition-colors hover:bg-slate-50 dark:hover:bg-slate-900/60",
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
                  "px-4 py-3 text-left text-sm font-semibold text-slate-600 dark:text-slate-200",
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
                  "px-4 py-3 align-top text-sm text-slate-700 dark:text-slate-300",
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
                  "font-semibold text-slate-900 dark:text-slate-100",
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
                  "text-slate-700 dark:text-slate-300",
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
                  "my-6 overflow-x-auto rounded-xl border border-slate-200 bg-slate-50 p-4 text-[13px] leading-6 text-slate-900 shadow-inner dark:border-slate-800 dark:bg-slate-950/90 dark:text-slate-100",
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
                  "my-8 w-full rounded-xl border border-slate-200 object-contain shadow-sm dark:border-slate-800",
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
            const normalizedLang = lang.toLowerCase();
            const childValue = Array.isArray(children)
              ? children.join("")
              : children;
            const sourceCode = typeof childValue === "string" ? childValue : "";

            if (normalizedLang === "youtube") {
              const embedId = sourceCode || "";
              return <Youtube embedId={embedId.trim()} />;
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

            if (inline || normalizedLang === "plaintext") {
              return (
                <code
                  {...rest}
                  className={mergeClassNames(
                    "rounded-md bg-slate-100 px-1.5 py-0.5 font-mono text-[13px] text-slate-700 dark:bg-slate-800/80 dark:text-slate-200",
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
              "not-prose mt-4 overflow-x-auto rounded-2xl border border-slate-200 bg-slate-50 px-4 py-4 text-[13px] leading-6 text-slate-900 shadow-sm dark:border-slate-800 dark:bg-slate-950/90 dark:text-slate-100",
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
