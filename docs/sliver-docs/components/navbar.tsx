"use client";

import { SliversIcon } from "@/components/icons/slivers";
import { useSearchContext } from "@/util/search-context";
import { Themes } from "@/util/themes";
import { faGithub } from "@fortawesome/free-brands-svg-icons";
import {
  faBars,
  faHome,
  faMoon,
  faSearch,
  faSun,
  faXmark,
} from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  Button,
  Input,
  Link,
  Navbar,
  NavbarBrand,
  NavbarContent,
  NavbarItem,
  Tooltip,
} from "@heroui/react";
import { useTheme } from "next-themes";
import { useRouter } from "next/router";
import React from "react";

export type TopNavbarProps = {};

export default function TopNavbar(props: TopNavbarProps) {
  const router = useRouter();
  const search = useSearchContext();
  const { theme, setTheme } = useTheme();
  const [mounted, setMounted] = React.useState(false);

  React.useEffect(() => {
    setMounted(true);
  }, []);

  const lightDarkModeIcon = React.useMemo(() => {
    if (!mounted) {
      return faMoon;
    }
    return theme === Themes.DARK ? faSun : faMoon;
  }, [mounted, theme]);

  const [query, setQuery] = React.useState("");
  const [isMobileMenuOpen, setIsMobileMenuOpen] = React.useState(false);

  React.useEffect(() => {
    setIsMobileMenuOpen(false);
  }, [router.pathname]);

  const handleSearchSubmit = React.useCallback(() => {
    if (query.trim().length === 0) {
      return;
    }
    router.push({ pathname: "/search/", query: { search: query } });
    setQuery("");
  }, [query, router]);

  const renderSearchInput = (wrapperClassName?: string) => (
    <div className={wrapperClassName}>
      <Tooltip content="Press enter to search" isOpen={query.length > 0}>
        <Input
          size="sm"
          placeholder="Search..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onClear={() => setQuery("")}
          startContent={<FontAwesomeIcon icon={faSearch} />}
          onKeyDown={(e) => {
            if (e.key === "Enter") {
              handleSearchSubmit();
            }
          }}
        />
      </Tooltip>
    </div>
  );

  return (
    <>
      <Navbar
        isBordered
        maxWidth="full"
        classNames={{
          item: [
            "flex",
          "relative",
          "h-full",
          "items-center",
          "data-[active=true]:after:content-['']",
          "data-[active=true]:after:absolute",
          "data-[active=true]:after:bottom-0",
          "data-[active=true]:after:left-0",
          "data-[active=true]:after:right-0",
          "data-[active=true]:after:h-[2px]",
          "data-[active=true]:after:rounded-[2px]",
          "data-[active=true]:after:bg-primary",
        ],
      }}
    >
      <NavbarBrand>
        <SliversIcon />
        <span className="hidden sm:block font-bold text-inherit">
          &nbsp; Sliver C2
        </span>
      </NavbarBrand>

      <NavbarContent className="hidden md:flex">
        <NavbarItem isActive={router.pathname === "/"}>
          <Button
            variant="light"
            color={router.pathname === "/" ? "primary" : "default"}
            as={Link}
            onPress={() => router.push("/")}
          >
            <FontAwesomeIcon fixedWidth icon={faHome} />
          </Button>
        </NavbarItem>

        <NavbarItem isActive={router.pathname.startsWith("/tutorials")}>
          <Button
            variant="light"
            color={router.pathname === "/tutorials" ? "primary" : "default"}
            as={Link}
            onPress={() => router.push("/tutorials")}
          >
            Tutorials
          </Button>
        </NavbarItem>

        <NavbarItem isActive={router.pathname.startsWith("/docs")}>
          <Button
            variant="light"
            color={router.pathname === "/docs" ? "primary" : "default"}
            as={Link}
            onPress={() => router.push("/docs")}
          >
            Docs
          </Button>
        </NavbarItem>
      </NavbarContent>

      <NavbarContent as="div" justify="end" className="hidden md:flex">
        {renderSearchInput("w-64")}
        <Button
          variant="ghost"
          onPress={() => {
            setTheme(theme === Themes.DARK ? Themes.LIGHT : Themes.DARK);
          }}
          isDisabled={!mounted}
        >
          <FontAwesomeIcon icon={lightDarkModeIcon} />
        </Button>

        <Button
          variant="ghost"
          onPress={() => {
            window.open("https://github.com/BishopFox/sliver", "_blank");
          }}
        >
          <FontAwesomeIcon icon={faGithub} />
        </Button>
      </NavbarContent>

      <div className="flex items-center gap-2 md:hidden">
        <Button
          variant="ghost"
          onPress={() => {
            setTheme(theme === Themes.DARK ? Themes.LIGHT : Themes.DARK);
          }}
          isDisabled={!mounted}
        >
          <FontAwesomeIcon icon={lightDarkModeIcon} />
        </Button>
        <Button
          variant="ghost"
          aria-label={isMobileMenuOpen ? "Close menu" : "Open menu"}
          onPress={() => setIsMobileMenuOpen((current) => !current)}
        >
          <FontAwesomeIcon icon={isMobileMenuOpen ? faXmark : faBars} />
        </Button>
      </div>
    </Navbar>

      {isMobileMenuOpen ? (
        <div className="md:hidden border-b border-default-200 dark:border-default-100/40 bg-content1 px-4 pb-4 shadow-sm">
          <div className="mt-3">{renderSearchInput("w-full")}</div>
          <div className="mt-4 flex flex-col gap-2">
            <Button
              variant={router.pathname === "/" ? "solid" : "light"}
              color={router.pathname === "/" ? "primary" : "default"}
              onPress={() => {
                router.push("/");
                setIsMobileMenuOpen(false);
              }}
              startContent={<FontAwesomeIcon fixedWidth icon={faHome} />}
            >
              Home
            </Button>
            <Button
              variant={router.pathname.startsWith("/tutorials") ? "solid" : "light"}
              color={router.pathname.startsWith("/tutorials") ? "primary" : "default"}
              onPress={() => {
                router.push("/tutorials");
                setIsMobileMenuOpen(false);
              }}
            >
              Tutorials
            </Button>
            <Button
              variant={router.pathname.startsWith("/docs") ? "solid" : "light"}
              color={router.pathname.startsWith("/docs") ? "primary" : "default"}
              onPress={() => {
                router.push("/docs");
                setIsMobileMenuOpen(false);
              }}
            >
              Docs
            </Button>
            <Button
              variant="light"
              onPress={() => {
                window.open("https://github.com/BishopFox/sliver", "_blank");
              }}
              startContent={<FontAwesomeIcon icon={faGithub} />}
            >
              GitHub
            </Button>
          </div>
        </div>
      ) : null}
    </>
  );
}
