# Sliver Docs

The source code for the Sliver documentation site.

### Developers

To run the documentation site locally first install the dependencies and then run the `dev` npm script:

```bash
npm install
npm run dev
```

**NOTE:** The markdown is compiled into a static JSON object at build time. This means if you edit a `.md` file you will need to restart the dev server to see your changes.

### Offline Docs

To run your own copy of the documentation first install the dependencies and then run the `offline` npm script:

```bash
npm install
npm run offline
```

This will produce a `www.zip` file that contains the static html and JavaScript for the documentation site. You can extract this archive and host it anywhere:

```bash
unzip -o www.zip
cd out/
python -m http.server 8000
```

Then open your browser to `http://localhost:8000/` to view the documentation.
