import Navbar from "@/components/navbar";
import "@/styles/globals.css";
import { Themes } from "@/util/themes";
import { NextUIProvider } from "@nextui-org/react";
import { ThemeProvider as NextThemesProvider, useTheme } from "next-themes";
import type { AppProps } from "next/app";

export default function App({ Component, pageProps }: AppProps) {
  const { theme, setTheme } = useTheme();
  function getThemeState(): Themes {
    if (typeof window !== "undefined") {
      const loadedTheme = localStorage.getItem("theme");
      const currentTheme = loadedTheme ? (loadedTheme as Themes) : Themes.DARK;
      setTheme(currentTheme);
    }
    return Themes.DARK;
  }

  return (
    <NextUIProvider>
      <NextThemesProvider attribute="class" defaultTheme={getThemeState()}>
        <Navbar />
        <Component {...pageProps} />
      </NextThemesProvider>
    </NextUIProvider>
  );
}
