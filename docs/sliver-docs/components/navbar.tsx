"use client";

import { SliversIcon } from "@/components/icons/slivers";
import { Themes } from "@/util/themes";
import { faDiscord, faGithub } from "@fortawesome/free-brands-svg-icons";
import {
  faBars,
  faHome,
  faMoon,
  faSearch,
  faSun,
  faXmark,
} from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { Button, SearchField, Tooltip } from "@heroui/react";
import { useTheme } from "next-themes";
import { useRouter } from "next/router";
import React from "react";

const routes = [
  { href: "/tutorials", label: "Tutorials" },
  { href: "/talks", label: "Talks" },
  { href: "/docs", label: "Docs" },
];

export default function TopNavbar() {
  const router = useRouter();
  const { theme, setTheme } = useTheme();
  const [query, setQuery] = React.useState("");
  const [isMobileMenuOpen, setIsMobileMenuOpen] = React.useState(false);

  const activeTheme = theme || Themes.DARK;
  const lightDarkModeIcon = React.useMemo(() => {
    return activeTheme === Themes.DARK ? faSun : faMoon;
  }, [activeTheme]);

  const handleSearchSubmit = React.useCallback(
    (value = query) => {
      const searchQuery = value.trim();

      if (searchQuery.length === 0) {
        return;
      }
      router.push({ pathname: "/search/", query: { search: searchQuery } });
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

  const renderSearchInput = (wrapperClassName?: string) => (
    <SearchField
      aria-label="Search documentation"
      className={wrapperClassName}
      fullWidth
      value={query}
      onChange={setQuery}
      onSubmit={handleSearchSubmit}
    >
      <SearchField.Group>
        <SearchField.SearchIcon>
          <FontAwesomeIcon icon={faSearch} />
        </SearchField.SearchIcon>
        <SearchField.Input placeholder="Search..." />
        {query.length > 0 ? <SearchField.ClearButton aria-label="Clear search" /> : null}
      </SearchField.Group>
    </SearchField>
  );

  const renderDesktopRoute = (href: string, label: string) => {
    const isActive = router.pathname.startsWith(href);

    return (
      <li key={href}>
        <Button
          variant={isActive ? "secondary" : "ghost"}
          onPress={() => router.push(href)}
        >
          {label}
        </Button>
      </li>
    );
  };

  const renderMobileRoute = (href: string, label: string) => {
    const isActive = href === "/" ? router.pathname === "/" : router.pathname.startsWith(href);

    return (
      <Button
        key={href}
        fullWidth
        variant={isActive ? "primary" : "ghost"}
        onPress={() => {
          router.push(href);
          setIsMobileMenuOpen(false);
        }}
      >
        {href === "/" ? <FontAwesomeIcon fixedWidth icon={faHome} /> : null}
        {label}
      </Button>
    );
  };

  return (
    <>
      <nav className="sliver-navbar-glass sticky top-0 z-40 px-4 shadow-sm md:px-6">
        <div className="flex h-16 items-center gap-4">
          <button
            type="button"
            className="flex items-center gap-2 text-foreground"
            onClick={() => {
              router.push("/");
              setIsMobileMenuOpen(false);
            }}
            aria-label="Sliver C2 home"
          >
            <SliversIcon />
            <span className="hidden font-bold text-inherit sm:block">
              Sliver C2
            </span>
          </button>

          <ul className="hidden items-center gap-2 md:flex">
            <li>
              {renderTooltip(
                "Home",
                <Button
                  isIconOnly
                  aria-label="Home"
                  variant={router.pathname === "/" ? "secondary" : "ghost"}
                  onPress={() => router.push("/")}
                >
                  <FontAwesomeIcon fixedWidth icon={faHome} />
                </Button>,
              )}
            </li>
            {routes.map((route) => renderDesktopRoute(route.href, route.label))}
          </ul>

          <div className="ml-auto hidden items-center gap-2 md:flex">
            {renderSearchInput("w-64")}
            {renderTooltip(
              activeTheme === Themes.DARK ? "Switch to light mode" : "Switch to dark mode",
              <Button
                isIconOnly
                aria-label={activeTheme === Themes.DARK ? "Switch to light mode" : "Switch to dark mode"}
                variant="ghost"
                onPress={() => {
                  setTheme(activeTheme === Themes.DARK ? Themes.LIGHT : Themes.DARK);
                }}
              >
                <FontAwesomeIcon icon={lightDarkModeIcon} />
              </Button>,
            )}

            {renderTooltip(
              "Join Discord",
              <Button
                isIconOnly
                aria-label="Join Discord"
                variant="ghost"
                onPress={() => {
                  window.open(
                    "https://discord.com/channels/791066041198968873/1339996286514106409",
                    "_blank",
                    "noopener,noreferrer",
                  );
                }}
              >
                <FontAwesomeIcon icon={faDiscord} />
              </Button>,
            )}

            {renderTooltip(
              "View on GitHub",
              <Button
                isIconOnly
                aria-label="View on GitHub"
                variant="ghost"
                onPress={() => {
                  window.open(
                    "https://github.com/BishopFox/sliver",
                    "_blank",
                    "noopener,noreferrer",
                  );
                }}
              >
                <FontAwesomeIcon icon={faGithub} />
              </Button>,
            )}
          </div>

          <div className="ml-auto flex items-center gap-2 md:hidden">
            {renderTooltip(
              activeTheme === Themes.DARK ? "Switch to light mode" : "Switch to dark mode",
              <Button
                isIconOnly
                aria-label={activeTheme === Themes.DARK ? "Switch to light mode" : "Switch to dark mode"}
                variant="ghost"
                onPress={() => {
                  setTheme(activeTheme === Themes.DARK ? Themes.LIGHT : Themes.DARK);
                }}
              >
                <FontAwesomeIcon icon={lightDarkModeIcon} />
              </Button>,
            )}
            {renderTooltip(
              isMobileMenuOpen ? "Close menu" : "Open menu",
              <Button
                isIconOnly
                variant="ghost"
                aria-label={isMobileMenuOpen ? "Close menu" : "Open menu"}
                onPress={() => setIsMobileMenuOpen((current) => !current)}
              >
                <FontAwesomeIcon icon={isMobileMenuOpen ? faXmark : faBars} />
              </Button>,
            )}
          </div>
        </div>
      </nav>

      {isMobileMenuOpen ? (
        <div className="sliver-navbar-glass px-4 pb-4 shadow-sm md:hidden">
          <div className="mt-3">{renderSearchInput("w-full")}</div>
          <div className="mt-4 flex flex-col gap-2">
            {renderMobileRoute("/", "Home")}
            {routes.map((route) => renderMobileRoute(route.href, route.label))}
            <Button
              fullWidth
              variant="ghost"
              onPress={() => {
                window.open(
                  "https://discord.com/channels/791066041198968873/1339996286514106409",
                  "_blank",
                  "noopener,noreferrer",
                );
              }}
            >
              <FontAwesomeIcon icon={faDiscord} />
              Discord
            </Button>
            <Button
              fullWidth
              variant="ghost"
              onPress={() => {
                window.open(
                  "https://github.com/BishopFox/sliver",
                  "_blank",
                  "noopener,noreferrer",
                );
              }}
            >
              <FontAwesomeIcon icon={faGithub} />
              GitHub
            </Button>
          </div>
        </div>
      ) : null}
    </>
  );
}
