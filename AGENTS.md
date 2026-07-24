<!--
SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>

SPDX-License-Identifier: Apache-2.0
-->

# Agent notes

## REUSE licensing

This project follows the [REUSE](https://reuse.software/) specification with
Apache License 2.0. Keep files compliant when adding or editing code.

### Lint

Check compliance from the repository root:

```bash
reuse lint
```

### Annotate

Add copyright and license headers to files:

```bash
reuse annotate \
  --copyright="Robert Eggl <robert@eggl.dev>" \
  --license=Apache-2.0 \
  --year=2026 \
  path/to/file
```

For a directory tree:

```bash
reuse annotate \
  --copyright="Robert Eggl <robert@eggl.dev>" \
  --license=Apache-2.0 \
  --year=2026 \
  --recursive \
  --skip-existing \
  --fallback-dot-license \
  path/to/dir
```

- `--skip-existing` leaves files that already have REUSE info alone.
- `--fallback-dot-license` writes a `.license` sidecar for unrecognized comment styles.
