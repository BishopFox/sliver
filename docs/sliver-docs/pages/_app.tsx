import Navbar from "@/components/navbar";
import "@/styles/globals.css";
import { Themes } from "@/util/themes";
import { NextUIProvider } from "@nextui-org/react";
import {
  HydrationBoundary,
  QueryClient,
  QueryClientProvider,
} from "@tanstack/react-query";
import { ThemeProvider as NextThemesProvider, useTheme } from "next-themes";
import type { AppProps } from "next/app";
import React from "react";

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

  const [queryClient] = React.useState(() => new QueryClient());

  return (
    <NextUIProvider>
      <NextThemesProvider attribute="class" defaultTheme={getThemeState()}>
        <QueryClientProvider client={queryClient}>
          <HydrationBoundary state={pageProps.dehydratedState}>
            <Navbar />
            <Component {...pageProps} />
          </HydrationBoundary>
        </QueryClientProvider>
      </NextThemesProvider>
    </NextUIProvider>
  );
}
