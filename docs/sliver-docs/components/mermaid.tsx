import { Themes } from "@/util/themes";
import { useTheme } from "next-themes";
import React from "react";

type MermaidProps = {
  diagram: string;
  minHeight?: number;
};

const Mermaid = ({ diagram, minHeight }: MermaidProps) => {
  const { theme } = useTheme();
  const containerRef = React.useRef<HTMLDivElement>(null);
  const [error, setError] = React.useState<string | null>(null);
  const normalizedMinHeight =
    Number.isFinite(minHeight) && minHeight
      ? Math.min(1200, Math.max(120, Math.floor(minHeight)))
      : undefined;

  React.useEffect(() => {
    let cancelled = false;
    const target = containerRef.current;
    const source = diagram.trim();

    if (!target || !source) {
      return;
    }

    const renderDiagram = async () => {
      try {
        const mermaidModule = await import("mermaid");
        const mermaid = mermaidModule.default;
        const mermaidTheme = theme === Themes.LIGHT ? "default" : "dark";

        mermaid.initialize({
          startOnLoad: false,
          securityLevel: "strict",
          theme: mermaidTheme,
        });

        const graphId = `sliver-mermaid-${Math.random()
          .toString(36)
          .slice(2, 10)}`;
        const { svg, bindFunctions } = await mermaid.render(graphId, source);

        if (cancelled || !containerRef.current) {
          return;
        }

        containerRef.current.innerHTML = svg;
        bindFunctions?.(containerRef.current);

        if (normalizedMinHeight) {
          const svgEl = containerRef.current.querySelector("svg");
          if (svgEl) {
            const viewBox = svgEl.getAttribute("viewBox") || "";
            const parts = viewBox
              .split(/\s+/)
              .map((part) => Number.parseFloat(part))
              .filter((value) => Number.isFinite(value));

            let naturalWidth = 0;
            let naturalHeight = 0;
            if (parts.length === 4) {
              naturalWidth = parts[2] || 0;
              naturalHeight = parts[3] || 0;
            }

            if ((naturalWidth <= 0 || naturalHeight <= 0) && svgEl.getBBox) {
              try {
                const bounds = svgEl.getBBox();
                naturalWidth = bounds.width;
                naturalHeight = bounds.height;
              } catch {
                // No-op: leave natural dimensions unset if bbox is unavailable.
              }
            }

            if (naturalWidth > 0 && naturalHeight > 0 && naturalHeight < normalizedMinHeight) {
              const scale = normalizedMinHeight / naturalHeight;
              const targetWidth = Math.round(naturalWidth * scale);
              const targetHeight = Math.round(naturalHeight * scale);

              svgEl.setAttribute("width", `${targetWidth}`);
              svgEl.setAttribute("height", `${targetHeight}`);
              svgEl.style.width = `${targetWidth}px`;
              svgEl.style.height = `${targetHeight}px`;
              svgEl.style.maxWidth = "none";
            }
          }
        }

        setError(null);
      } catch (renderError) {
        if (cancelled || !containerRef.current) {
          return;
        }
        containerRef.current.innerHTML = "";
        const message =
          renderError instanceof Error
            ? renderError.message
            : "Unknown Mermaid rendering error";
        setError(message);
      }
    };

    void renderDiagram();

    return () => {
      cancelled = true;
      if (target) {
        target.innerHTML = "";
      }
    };
  }, [diagram, normalizedMinHeight, theme]);

  if (error) {
    return (
      <div className="not-prose my-6 text-sm text-rose-700 dark:text-rose-300">
        Failed to render Mermaid diagram: {error}
      </div>
    );
  }

  return (
    <div className="not-prose my-6 overflow-x-auto">
      <div
        className="mx-auto flex w-fit min-w-max items-center justify-center"
        style={normalizedMinHeight ? { minHeight: normalizedMinHeight } : undefined}
      >
        <div
          ref={containerRef}
          className="min-w-max [&_svg]:h-auto [&_svg]:max-w-none"
        />
      </div>
    </div>
  );
};

export default Mermaid;
