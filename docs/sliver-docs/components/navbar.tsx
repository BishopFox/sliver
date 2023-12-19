"use client";

import { SliversIcon } from "@/components/icons/slivers";
import { useSearchContext } from "@/util/search-context";
import { Themes } from "@/util/themes";
import { faGithub } from "@fortawesome/free-brands-svg-icons";
import {
  faHome,
  faMoon,
  faSearch,
  faSun,
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
} from "@nextui-org/react";
import { useTheme } from "next-themes";
import { useRouter } from "next/router";
import React from "react";

export type TopNavbarProps = {};

export default function TopNavbar(props: TopNavbarProps) {
  const router = useRouter();
  const search = useSearchContext();
  const { theme, setTheme } = useTheme();
  const lightDarkModeIcon = React.useMemo(
    () => (theme === Themes.DARK ? faSun : faMoon),
    [theme]
  );

  const [query, setQuery] = React.useState("");

  return (
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

      <NavbarContent>
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

      <NavbarContent as="div" justify="end">
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
                router.push({ pathname: `/search/`, query: { search: query } });
                setQuery("");
              }
            }}
          />
        </Tooltip>

        <Button
          variant="ghost"
          onPress={() => {
            setTheme(theme === Themes.DARK ? Themes.LIGHT : Themes.DARK);
          }}
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
    </Navbar>
  );
}
