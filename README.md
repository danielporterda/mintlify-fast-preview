# mintlify-fast-preview

`mintlify-fast-preview` is a clean-room Go preview server for Mintlify-style docs
sites. The goal is fast local iteration for docs repositories that already carry
`docs.json`, MDX pages, static assets, and custom CSS.

This project intentionally starts with a red golden comparison suite. Run:

```bash
go test ./...
```

The golden suite should initially report `7/7 golden tests failed`. Unit tests
cover the parser, router, renderer primitives, search index, asset resolution,
and watcher event coalescing while page parity is moved forward incrementally.

Planned CLI:

```bash
mintfast dev --root ./docs-main --port 3000 --host 127.0.0.1 --no-open
mintfast validate --root ./docs-main
mintfast render --root ./docs-main --out ./dist
```

The implementation must not copy Mintlify internals or depend on Mintlify's
shared `~/.mintlify` runtime.

