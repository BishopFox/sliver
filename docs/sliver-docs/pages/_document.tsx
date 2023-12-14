import { Head, Html, Main, NextScript } from "next/document";

// This generates the https: and wss: "connect-src" directives based on the above backends list so its a little easier to edit.
const CSP = `default-src 'none'; script-src 'self' 'unsafe-eval'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; font-src 'self'; connect-src 'self'`;

export default function Document() {
  return (
    <Html lang="en">
      <Head>
        <Head>
          <meta httpEquiv="Content-Security-Policy" content={CSP} />
        </Head>
      </Head>
      <body>
        <Main />
        <NextScript />
      </body>
    </Html>
  );
}
