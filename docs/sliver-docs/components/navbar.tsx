"use client";

import { SliversIcon } from "@/components/icons/slivers";
import { Themes } from "@/util/themes";
import { faDiscord, faGithub } from "@fortawesome/free-brands-svg-icons";
import {
  faBars,
  faMoon,
  faSearch,
  faSun,
  faXmark,
} from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { Button, Drawer, SearchField, Tooltip } from "@heroui/react";
import { useTheme } from "next-themes";
import NextLink from "next/link";
import { useRouter } from "next/router";
import React from "react";

const routes = [
  { href: "/docs", label: "Docs" },
  { href: "/tutorials", label: "Tutorials" },
  { href: "/talks", label: "Talks" },
];

const discordUrl =
  "https://discord.com/channels/791066041198968873/1339996286514106409";
const githubUrl = "https://github.com/BishopFox/sliver";

export default function TopNavbar() {
  const router = useRouter();
  const { theme, setTheme } = useTheme();
  const [query, setQuery] = React.useState("");
  const [isMobileMenuOpen, setIsMobileMenuOpen] = React.useState(false);

  const activeTheme = theme || Themes.DARK;
  const isDarkTheme = activeTheme === Themes.DARK;
  const lightDarkModeIcon = isDarkTheme ? faSun : faMoon;
  const themeToggleLabel = isDarkTheme
    ? "Switch to light mode"
    : "Switch to dark mode";

  const toggleTheme = React.useCallback(() => {
    setTheme(isDarkTheme ? Themes.LIGHT : Themes.DARK);
  }, [isDarkTheme, setTheme]);

  React.useEffect(() => {
    const desktopBreakpoint = window.matchMedia("(min-width: 768px)");
    const closeMobileMenu = (event: MediaQueryListEvent) => {
      if (event.matches) {
        setIsMobileMenuOpen(false);
      }
    };

    desktopBreakpoint.addEventListener("change", closeMobileMenu);
    return () =>
      desktopBreakpoint.removeEventListener("change", closeMobileMenu);
  }, []);

  const handleSearchSubmit = React.useCallback(
    (value = query) => {
      const searchQuery = value.trim();

      if (searchQuery.length === 0) {
        return;
      }

      void router.push({
        pathname: "/search/",
        query: { search: searchQuery },
      });
      setQuery("");
      setIsMobileMenuOpen(false);
    },
    [query, router],
  );

  const renderTooltip = React.useCallback(
    (content: string, children: React.ReactNode) => (
      <Tooltip delay={0}>
        {children}
        <Tooltip.Content>{content}</Tooltip.Content>
      </Tooltip>
    ),
    [],
  );

  const renderSearchInput = (className?: string) => (
    <SearchField
      aria-label="Search documentation"
      className={className}
      fullWidth
      value={query}
      variant="secondary"
      onChange={setQuery}
      onSubmit={handleSearchSubmit}
    >
      <SearchField.Group>
        <SearchField.SearchIcon>
          <FontAwesomeIcon icon={faSearch} />
        </SearchField.SearchIcon>
        <SearchField.Input placeholder="Search documentation…" />
        {query.length > 0 ? (
          <SearchField.ClearButton aria-label="Clear search" />
        ) : null}
      </SearchField.Group>
    </SearchField>
  );

  const renderDesktopRoute = (href: string, label: string) => {
    const isActive = router.pathname.startsWith(href);

    return (
      <li key={href}>
        <NextLink
          href={href}
          aria-current={isActive ? "page" : undefined}
          className={`inline-flex h-10 items-center justify-center rounded-xl px-4 text-sm font-medium no-underline outline-none focus-visible:ring-2 focus-visible:ring-accent/40 ${
            isActive
              ? "bg-surface-secondary text-foreground shadow-surface"
              : "text-muted hover:bg-surface-secondary hover:text-foreground"
          }`}
        >
          {label}
        </NextLink>
      </li>
    );
  };

  const renderMobileRoute = (href: string, label: string) => {
    const isActive =
      href === "/"
        ? router.pathname === "/"
        : router.pathname.startsWith(href);

    return (
      <NextLink
        key={href}
        href={href}
        aria-current={isActive ? "page" : undefined}
        className={`inline-flex min-h-10 w-full items-center rounded-xl px-3 py-2 text-sm font-medium no-underline outline-none focus-visible:ring-2 focus-visible:ring-accent/40 ${
          isActive
            ? "bg-surface-secondary text-foreground shadow-surface"
            : "text-muted hover:bg-surface-secondary hover:text-foreground"
        }`}
        onClick={() => setIsMobileMenuOpen(false)}
      >
        {label}
      </NextLink>
    );
  };

  return (
    <>
      <nav
        aria-label="Primary navigation"
        className="sticky top-0 z-40 h-16 border-b border-separator bg-background/85 backdrop-blur-xl"
      >
        <div className="mx-auto flex h-16 w-full max-w-[1440px] items-center gap-3 px-4 sm:px-6 lg:px-8">
          <NextLink
            href="/"
            aria-label="Sliver documentation home"
            className="-ml-2 inline-flex h-10 shrink-0 items-center gap-2 rounded-xl px-2 text-sm font-semibold text-foreground no-underline outline-none hover:bg-surface-secondary focus-visible:ring-2 focus-visible:ring-accent/40"
          >
            <span className="flex size-8 items-center justify-center rounded-xl bg-accent text-accent-foreground">
              <SliversIcon height={18} />
            </span>
            <span className="hidden items-baseline gap-1.5 sm:flex">
              <span>Sliver</span>
              <span className="font-normal text-muted">Docs</span>
            </span>
          </NextLink>

          <ul className="hidden items-center gap-1 md:flex">
            {routes.map((route) =>
              renderDesktopRoute(route.href, route.label),
            )}
          </ul>

          <div className="ml-auto hidden items-center gap-1 md:flex">
            {renderSearchInput("mr-2 w-56 xl:w-72")}

            {renderTooltip(
              themeToggleLabel,
              <Button
                isIconOnly
                aria-label={themeToggleLabel}
                variant="ghost"
                onPress={toggleTheme}
              >
                <FontAwesomeIcon icon={lightDarkModeIcon} />
              </Button>,
            )}

            {renderTooltip(
              "Join Discord",
              <a
                href={discordUrl}
                target="_blank"
                rel="noreferrer"
                aria-label="Join Discord"
                className="inline-flex size-10 items-center justify-center rounded-xl text-muted no-underline outline-none hover:bg-surface-secondary hover:text-foreground focus-visible:ring-2 focus-visible:ring-accent/40"
              >
                <FontAwesomeIcon icon={faDiscord} />
              </a>,
            )}

            {renderTooltip(
              "View on GitHub",
              <a
                href={githubUrl}
                target="_blank"
                rel="noreferrer"
                aria-label="View on GitHub"
                className="inline-flex size-10 items-center justify-center rounded-xl text-muted no-underline outline-none hover:bg-surface-secondary hover:text-foreground focus-visible:ring-2 focus-visible:ring-accent/40"
              >
                <FontAwesomeIcon icon={faGithub} />
              </a>,
            )}
          </div>

          <div className="ml-auto flex items-center gap-1 md:hidden">
            {renderTooltip(
              themeToggleLabel,
              <Button
                isIconOnly
                aria-label={themeToggleLabel}
                variant="ghost"
                onPress={toggleTheme}
              >
                <FontAwesomeIcon icon={lightDarkModeIcon} />
              </Button>,
            )}

            {renderTooltip(
              isMobileMenuOpen ? "Close menu" : "Open menu",
              <Button
                isIconOnly
                aria-controls="mobile-navigation"
                aria-expanded={isMobileMenuOpen}
                aria-label={isMobileMenuOpen ? "Close menu" : "Open menu"}
                variant="ghost"
                onPress={() =>
                  setIsMobileMenuOpen((current) => !current)
                }
              >
                <FontAwesomeIcon icon={isMobileMenuOpen ? faXmark : faBars} />
              </Button>,
            )}
          </div>
        </div>
      </nav>

      <Drawer.Backdrop
        isOpen={isMobileMenuOpen}
        variant="blur"
        onOpenChange={setIsMobileMenuOpen}
      >
        <Drawer.Content placement="right">
          <Drawer.Dialog
            id="mobile-navigation"
            className="w-[min(24rem,calc(100vw-1.5rem))]"
          >
            <Drawer.CloseTrigger />
            <Drawer.Header>
              <Drawer.Heading>Navigate Sliver Docs</Drawer.Heading>
            </Drawer.Header>
            <Drawer.Body className="flex flex-col gap-5 pt-2">
              {renderSearchInput("w-full")}

              <nav
                aria-label="Mobile navigation"
                className="flex flex-col gap-1"
              >
                {renderMobileRoute("/", "Home")}
                {routes.map((route) =>
                  renderMobileRoute(route.href, route.label),
                )}
              </nav>
            </Drawer.Body>
            <Drawer.Footer className="grid grid-cols-2 gap-2">
              <a
                href={discordUrl}
                target="_blank"
                rel="noreferrer"
                className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-xl border border-separator px-4 text-sm font-medium text-foreground no-underline outline-none hover:bg-surface-secondary focus-visible:ring-2 focus-visible:ring-accent/40"
              >
                <FontAwesomeIcon icon={faDiscord} />
                Discord
              </a>
              <a
                href={githubUrl}
                target="_blank"
                rel="noreferrer"
                className="inline-flex h-10 w-full items-center justify-center gap-2 rounded-xl border border-separator px-4 text-sm font-medium text-foreground no-underline outline-none hover:bg-surface-secondary focus-visible:ring-2 focus-visible:ring-accent/40"
              >
                <FontAwesomeIcon icon={faGithub} />
                GitHub
              </a>
            </Drawer.Footer>
          </Drawer.Dialog>
        </Drawer.Content>
      </Drawer.Backdrop>
    </>
  );
}
