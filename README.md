# mintlify-fast-preview

`mintlify-fast-preview` is a clean-room Go preview server for Mintlify-style docs
sites. The goal is fast local iteration for docs repositories that already carry
`docs.json`, MDX pages, static assets, and custom CSS.

The project started with a red golden comparison suite and now has the first
green baseline for the fixture docs site. Run:

```bash
go test ./...
```

Unit tests cover the parser, router, renderer primitives, search index, asset
resolution, live-reload broadcasting, and watcher event coalescing while page
parity is moved forward incrementally.

Planned CLI:

```bash
mintfast dev --root ./docs-main --port 3000 --host 127.0.0.1 --no-open
mintfast validate --root ./docs-main
mintfast render --root ./docs-main --out ./dist
```

## Current capabilities

- Parses `docs.json` dropdowns, versions, groups, string pages, and nested page
  group objects.
- Serves clean docs routes through a Mintlify-like shell with left navigation,
  recursive left navigation, right table-of-contents links, custom `styles.css`,
  dark-mode controls, copy buttons, and responsive CSS.
- Renders common MDX/Markdown used in the current docs tree: frontmatter,
  snippet imports, headings, paragraphs, lists, inline links/code, fenced code
  blocks, cards, columns, accordions, admonitions, JSX-style raw HTML attributes,
  generated reference HTML, and explicit compatibility panels for unsupported
  custom React components.
- Provides `/_mintfast/search?q=...` with a local offline index.
- Provides `/_mintfast/events` and polling live reload for docs/config/style
  file changes.
- Static export copies non-MDX assets alongside rendered pages.

## Nix consumption

Downstream docs repos can pin the tool as a flake input:

```nix
{
  inputs.mintlify-fast-preview.url = "github:danielporterda/mintlify-fast-preview";

  outputs = { self, nixpkgs, mintlify-fast-preview }:
    let
      system = "aarch64-darwin";
      pkgs = nixpkgs.legacyPackages.${system};
    in {
      devShells.${system}.default = pkgs.mkShell {
        packages = [
          mintlify-fast-preview.packages.${system}.default
        ];
      };
    };
}
```

Then run:

```bash
mintfast dev --root docs-main --port 3000 --host 127.0.0.1 --no-open
```

For one-off local validation without installing:

```bash
nix run github:danielporterda/mintlify-fast-preview -- \
  validate --root /path/to/docs-main
```

## Validation

Known-good local checks:

```bash
go test ./...
nix --extra-experimental-features 'nix-command flakes' build .# --no-link
mintfast validate --root /Users/danielporter/control/docs/docs-main
mintfast render --root /Users/danielporter/control/docs/docs-main --out /tmp/mintfast-real-docs
```

Browser smoke coverage starts a real `mintfast dev` server against a temporary
fixture-docs copy and checks clean routes, nav rendering, dark-mode persistence,
copy controls, mobile layout, and live reload:

```bash
nix-shell -p go python312Packages.playwright --run 'python tests/browser/preview_smoke.py'
```

Set `MINTFAST_CHROMIUM=/path/to/chromium` if Chromium is not discoverable on
`PATH`.

The current real docs tree validates and static-renders 1060 pages locally.

The implementation must not copy Mintlify internals or depend on Mintlify's
shared `~/.mintlify` runtime.
