# Repository Guidelines

## Project Structure & Module Organization
- `pages/` contains Next.js routes; Markdown docs live in `pages/docs/md`, tutorials under `pages/tutorials`.
- `components/` hosts reusable UI; `util/` stores loaders such as `Docs` and `Themes`.
- `public/` serves static assets and the generated `docs.json` consumed by search.
- `prebuild/` scripts run during `next build` to turn Markdown into JSON payloads.
- Tailwind config sits in `tailwind.config.js`; shared CSS lives in `styles/`.

## Build, Test, and Development Commands
- `npm install` sets up dependencies.
- `npm run dev` starts the docs server with hot reload on `http://localhost:3000`.
- `npm run build` exports the static site and regenerates `public/docs.json` plus tutorials JSON.
- `python3 -m http.server 8000 --directory out` serves the static build for smoke tests; `next start` does not support static exports.
- `npm run lint` runs ESLint.
- `npm run offline` builds, then zips `out/` for offline distribution.

## Coding Style & Naming Conventions
Use TypeScript with 2-space indentation and ES module imports. Keep React components PascalCase and helpers camelCase. Markdown filename stems are exact document names used by `?name=` links; follow existing title-case names and preserve filenames to avoid breaking links. Favor function components, hooks, and Tailwind utility classes; let ESLint autofix formatting issues and keep shared constants in `util/`.

## Testing Guidelines
No unit-test runner ships with this project, so treat `npm run lint` and `npm run build` as required gates. When adding interactive behavior, create targeted Playwright or Cypress checks under a `__tests__/` directory and describe how to run them in the PR. Always smoke-test the search panel locally after updating Markdown to confirm newly generated `docs.json` entries render.

## Commit & Pull Request Guidelines
History favors concise, imperative subjects (e.g., "Update workflows to use go v1.25.x"). Group related changes, reference issues with `#1234`, and avoid drive-by refactors. Pull requests should state purpose, call out UI diffs with screenshots or GIFs, confirm that `npm run lint` and `npm run build` both pass, and note any new content sources. Request a docs reviewer whenever Markdown copy changes.

## Content Authoring Tips
Edit or add docs in `pages/docs/md`; keep files flat so the generator exports clean slugs. After content updates, run `npm run build` to refresh `public/docs.json`, then spot-check sidebar headings in `npm run dev`. Tutorials follow the same workflow via `prebuild/generate-tutorials.js`, so align metadata fields to keep search results balanced.
