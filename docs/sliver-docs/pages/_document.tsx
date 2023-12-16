import Document, { Head, Html, Main, NextScript } from "next/document";
import React from "react";

// This generates the https: and wss: "connect-src" directives based on the above backends list so its a little easier to edit.
const CSP = `default-src 'none'; script-src 'self' 'unsafe-eval'; img-src 'self' data: https://user-images.githubusercontent.com https://i.imgur.com; style-src 'self' 'unsafe-inline'; font-src 'self'; connect-src 'self'`;

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
      <Html lang="en">
        <Head>
          <meta httpEquiv="Content-Security-Policy" content={CSP} />
        </Head>
        <body>
          <Main />
          <NextScript />
        </body>
      </Html>
    );
  }
}

export default MyDocument;
