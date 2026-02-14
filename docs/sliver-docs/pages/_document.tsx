import Document, { Head, Html, Main, NextScript } from "next/document";
import React from "react";

// This generates the https: and wss: "connect-src" directives based on the above backends list so its a little easier to edit.
const CSP = `default-src 'none'; script-src 'self' 'unsafe-eval' https://www.youtube.com https://s.ytimg.com; img-src 'self' data: https://i.ytimg.com https://*.ytimg.com; style-src 'self' 'unsafe-inline'; font-src 'self'; frame-src https://youtu.be https://www.youtube.com https://youtube.com https://www.youtube-nocookie.com; connect-src 'self'`;

class MyDocument extends Document {
  static async getInitialProps(ctx: any) {
    const initialProps = await Document.getInitialProps(ctx);
    return {
      ...initialProps,
      styles: React.Children.toArray([initialProps.styles]),
    };
  }

  render() {
    return (
      <Html lang="en" suppressHydrationWarning>
        <Head>
          <meta httpEquiv="Content-Security-Policy" content={CSP} />
        </Head>
        <body className="bg-background text-foreground">
          <Main />
          <NextScript />
        </body>
      </Html>
    );
  }
}

export default MyDocument;
