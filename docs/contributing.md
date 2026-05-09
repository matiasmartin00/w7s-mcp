---
layout: default
title: Contributing to Docs
nav_order: 6
---

# Contributing to Docs

The documentation lives in the `/docs` directory and is published via **GitHub Pages** using [Just the Docs](https://just-the-docs.com/).

---

## Adding or editing a page

1. Create or edit a `.md` file in `/docs/`
2. Add front matter at the top:

```yaml
---
layout: default
title: My Page Title
nav_order: 7          # controls sidebar order
---
```

3. Write content in standard Markdown
4. Open a pull request — GitHub Actions will rebuild and publish automatically on merge to `main`

---

## Local preview

To preview the docs site locally you need Ruby and Bundler:

```bash
cd docs
bundle install
bundle exec jekyll serve
# Open http://localhost:4000
```

If you don't have Ruby, you can use Docker:

```bash
docker run --rm -v "$PWD/docs:/srv/jekyll" -p 4000:4000 jekyll/jekyll jekyll serve
```

---

## Doc conventions

- **Keep docs aligned with implemented behavior.** Do not document speculative or planned features.
- Use the existing pages as structure references.
- Code examples should be minimal and runnable.
- Link to related pages using relative Markdown links (e.g. `[Tool Reference](tool-reference.md)`).

---

## Publish process

Docs are built and deployed automatically by `.github/workflows/docs.yml`:

1. Triggered on push to `main` that touches `docs/**` or `_config.yml`
2. Jekyll builds the site from `/docs`
3. The built site is deployed to the `gh-pages` branch via `actions/deploy-pages`

No manual steps are required after merging a PR.
