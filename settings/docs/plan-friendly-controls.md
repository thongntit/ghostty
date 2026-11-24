# Plan: friendly Ghostty option controls

Status: UX control registry and category-first navigation implemented; specialist coverage is still in progress
Date: 2026-09-20

## Product decision

Users should choose values through controls that match the option grammar. The
raw Ghostty value remains visible and available as an explicit escape hatch,
but it is not the default interaction for supported options.

Editors must be selected from explicit metadata or an authoritative provider.
The option name and description are not enough evidence to invent a control,
range, unit, choice list, or serializer. Open-ended values keep a raw editor by
design; known Ghostty grammars must not fall back to raw text merely because
they are represented as strings in the catalog.

## Current slice

The editor exposes every known catalog option across all loaded roots and
`config-file` includes. Each option is routed through a reviewed control model;
open-ended values intentionally keep a raw editor:

- category-first navigation: `Configured`, the reviewed setting categories,
  and an explicit `All Settings` escape hatch replace the initial flat catalog;
  global search still reaches every option.
- reviewed navigation areas: Application, Terminal, Shell, Clipboard, Window,
  Colors, Fonts, Keybindings, Mouse, GTK, Linux, and macOS, with unknown
  user-defined keys kept under Custom.

- the catalog now carries reviewed `control` and `group` metadata for all 207
  options; the runtime resolver remains backwards-compatible with older
  catalogs and inline test options.
- booleans: a visible inherit/off/on toggle; `ctrl+r` is the explicit raw
  escape hatch and accepting an untouched inherited value is a no-op.
- `font-size`: a metadata-declared half-point numeric stepper; `ctrl+r` opens
  raw numeric input.
- `background` and `foreground`: searchable named-color chooser with RGB
  values and a terminal swatch. Ghostty's `+list-colors` action supplies the
  inventory when available; a small fallback palette remains usable without
  Ghostty.
- `theme`: searchable chooser backed by Ghostty's `+list-themes` action when
  the installed binary is available. Unknown or custom values can be replaced
  through `ctrl+r` raw input.
- documented closed enums: searchable pickers generated from reviewed
  Ghostty metadata rather than inferred from option names.
- multi-value flags such as shell integration, bell features, font shaping,
  and notifications: each token has `default`, `enabled`, and `disabled`
  states, with `space` or `←/→` cycling and a raw fallback.
- durations: an amount field plus a unit picker for the supported Ghostty
  units; raw compound durations remain available through `ctrl+r`.
- paths: a directory browser for working directories and file paths, with a
  raw fallback for paths that are not present locally.
- fonts and actions: Ghostty's `+list-fonts` and `+list-actions` inventories
  power font-family, keybind, and command-palette pickers.
- structured repeatable forms: environment variables, font variations,
  codepoint maps, palette entries, keybindings, and command-palette entries
  are edited as fields instead of one opaque line.
- scalar structured forms: font variation fields and the
  `mouse-scroll-multiplier` precision/discrete pair no longer require users to
  remember Ghostty separators.
- font metric adjustments: `adjust-*` values use signed amount plus `px`/`%`
  unit controls, with raw fallback for unsupported syntax.
- resource limits: scrollback and Linux cgroup limits distinguish inherit,
  unlimited (where Ghostty supports it), and an exact numeric value.
- unknown tokens in reviewed multi-value lists are preserved when a known
  token is changed; the editor never silently drops newer Ghostty syntax.
- repeatable values and keybindings: occurrence list editors with add, edit,
  delete, clear, and source-file/line provenance.
- intentionally open-ended values: a validated raw-value editor remains
  available for shell commands, titles, environment values, custom namespaces,
  font feature expressions, arbitrary window coordinates, and custom shader
  details. These are content editors, not missing pickers.

Browse labels use human-readable names while retaining the raw key in the
details pane. Controls stage changes in memory. `ctrl+s` or quit confirmation
persists the candidate through conflict detection, backup, and atomic
replacement.

## Keyboard contract

- Category browse: `↑/k`, `↓/j` move, `enter/l` open, `/` global search,
  `c` configured, `a` all settings, `p` preview, `q` quit.
- Option browse: `↑/k`, `↓/j` move, `/` global search, `enter/e` edit,
  `esc/h` categories, `c` configured, `a` all settings, `g` categories,
  `r` reset, `u` revert, `p` preview, `q` quit.
- Choice editor: type to filter, `↑/↓` or `←/→` choose, `enter` select,
  `ctrl+r` raw value, `esc` cancel.
- Multi-select editor: `↑/↓` select a token, `space` or `←/→` cycle
  `default/enabled/disabled`, `enter` stage, `ctrl+r` raw value.
- Duration editor: type an amount, `←/→` choose a unit, `↑/↓` adjust the
  amount, `enter` stage, `ctrl+r` raw duration.
- Path editor: `↑/↓` browse, `enter` open/select, `s` select the current
  folder, `ctrl+r` raw path.
- Repeatable editor: `a` add, `e` edit, `d` delete, `r` clear, `enter` stage,
  `esc` cancel. Supported pairs and keybindings open field forms; `tab` or
  `enter` advances fields and action fields use a picker.
- Numeric editor: `←/→` adjust by the declared step, `ctrl+r` raw value,
  `enter` stage, `esc` cancel.
- The footer describes the valid keys for the active control.

## Next phases

1. Add subgroup-aware browsing and details for the large categories; the
   metadata and group headings are now available to support this without
   adding key-specific branches to the navigation model.
2. Add RGB channel controls for arbitrary colors, compound-duration rows, and
   guided editors for key capture, codepoint mappings, and font variation
   axes. Keep raw mode for values providers cannot safely describe.
3. Add explicit source-target selection for multi-file graphs when users want
   to edit a non-effective duplicate instead of the effective assignment.
4. Extend authoritative Ghostty validation to staged graphs with includes.
5. Make PTY flows a release gate: cover toggles, steppers, raw fallback,
   resize, cancel, validation failure, dirty quit, preview, save/reload, and
   conflicts using temporary fixtures.

## Acceptance criteria

- A supported single-file edit does not require Ghostty assignment syntax;
  only intentionally open-ended content uses text by default.
- Effective, source, target-local, and documented-default values remain
  distinct in the UI.
- Invalid existing values remain byte-preserved and can be explicitly
  replaced.
- Choice inventories are authoritative or clearly labeled as fallback
  presets; no option list is fabricated from a key name.
- Controls work with keyboard input at 80x24 and larger, without relying on
  color alone for meaning.
- Structured no-op edits preserve bytes, and unsupported/newer tokens remain
  visible and round-trip safely.
- Preview shows the exact candidate change and still performs no file write.
