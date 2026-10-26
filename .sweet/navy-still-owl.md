---
type: feature
breaking: true
---

Move embedding settings from MEANS.md to the machine-local Means user config.
On first use, create a commented, disabled config at
$XDG_CONFIG_HOME/means/config.yaml (default ~/.config/means/config.yaml) without
overwriting existing files. Settings apply to all collections; CLI reloads them
per invocation and MCP loads them at startup. Move legacy collection embedding
blocks into the user config and remove them from MEANS.md.
