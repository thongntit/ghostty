# Option schema

This directory contains the versioned option metadata used by the TUI.

The runtime schema must remain conservative: it can improve labels, grouping,
and editors, but the installed Ghostty binary remains authoritative when it is
available. Unknown configuration keys must stay editable and must survive a
round trip even when they are not present in this schema.

`cmd/schema-gen` imports documentation from Ghostty using
`ghostty +show-config --default --docs`. The fork's packaging script generates
and embeds the catalog from the executable in the app it is packaging.
Every generated option is exposed to the editor with
`control` and `group` metadata. Booleans, enums, numbers, colors, durations,
paths, multi-value flags, resource limits, metric adjustments, and structured
pairs use guided controls; repeatable and keybinding options use occurrence
lists and forms. Ghostty inventories drive fonts, themes, colors, and actions.
Open-ended values retain a validated raw editor by design, and unknown keys
remain editable as raw one-line values under Custom.

Standalone Go builds and tests use the checked-in fallback catalog. To refresh
it from a new Ghostty build, run this from the repository root:

```sh
cd settings
go run ./cmd/schema-gen --ghostty /path/to/ghostty --out internal/schema/options.json
```

The fork release workflow does this automatically in a temporary build directory.
To package a locally built fork app, run
`sh macos/bundle-settings.sh macos/build/Debug/Ghostty.app` from the repo root.

Do not redirect the full `+show-config --default --docs` output into a user
config file. It is documentation for the catalog, not a minimal config.
