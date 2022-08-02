# Docs

If you just want to _read_ the documentation, check out the [hosted version](https://docs.acorn.io)

## Run locally

To make changes to the docs and preview those changes locally:

```bash
cd docs
yarn # Run the first time, to install dependencies
yarn start
# http://localhost:3000/ opens up in your default browser, probably
```

## Naming files & directories

- All names should be entirely lowercase.  macOS filesystems are case-insensitive by default, Linux isn't.  Bad things happen.
- Names can be prefixed with `[a number]-` to force sort order.  The number isn't included in the output files/URLs.
- Prefer short, to the point names.  A single word when possible, hyphen-separated if you have to have mulitple.  These will become the URL for the file.
- Once published a file should never be renamed.  If you have to anyway, a redirect must be setup so that the old name continues to go somewhere.
- For files, a longer more descriptive title can go in the frontmatter (as `title:`).  These can be internationalized later.
- For directories, create a `_category_.yaml` with a `label:` in it to provide a longer display label/title.
