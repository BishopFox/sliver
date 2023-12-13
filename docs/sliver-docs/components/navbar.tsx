import { Themes } from "@/util/themes";
import { faEye, faHome } from "@fortawesome/free-solid-svg-icons";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import {
  Button,
  Link,
  Navbar,
  NavbarBrand,
  NavbarContent,
  NavbarItem,
} from "@nextui-org/react";
import { useTheme } from "next-themes";
import { useRouter } from "next/router";
import { SliversIcon } from "./icons/slivers";

export type TopNavbarProps = {};

export default function TopNavbar(props: TopNavbarProps) {
  const router = useRouter();
  const { theme, setTheme } = useTheme();

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
        <p className="hidden sm:block font-bold text-inherit">
          &nbsp; Sliver C2
        </p>
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
      </NavbarContent>

      <NavbarContent as="div" justify="end">
        <Button
          variant="light"
          onPress={() => {
            setTheme(theme === Themes.DARK ? Themes.LIGHT : Themes.DARK);
          }}
        >
          <FontAwesomeIcon icon={faEye} />
        </Button>
      </NavbarContent>
    </Navbar>
  );
}
