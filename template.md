# template.md — design cleanup for the auth-client template

> Notes from building **menu** on top of the `example/` auth-client template. Goal: every
> app built on the template should share the same "glassy" look out of the box — neutral
> white-translucent glass over a background image, mono type, green status accents, **no
> purple**. This file lists what the stock template ships that looks dated ("rudiments"),
> and how to fix it *in the template itself* so apps don't each need local overrides.

---

## The target style (the "menu look")

- Frosted **glass surfaces** (panels, cards, modals, buttons) = translucent dark fill +
  `backdrop-filter: blur()` over a blurred background image.
- **Neutral** palette: white at varying opacity for borders/text, never coloured chrome.
- Colour used **only** for meaning: green/amber/red status, red for destructive (logout).
- Monospace UI font (already in the template).

### 1. Palette — drop-in `:root` (replaces the purple theme)

The stock template `:root` is a **purple theme**. Replace these values:

```css
/* ❌ stock (purple) */
--border:        #252340;
--border-active: #7c68d0;
--text:          #b0b0cc;
--text-dim:      #42405e;
--accent:        #a78bfa;
--neon:          #c4b5fd;

/* ✅ neutral glass */
--border:        rgba(255, 255, 255, 0.14);
--border-active: rgba(255, 255, 255, 0.42);
--text:          rgba(255, 255, 255, 0.82);
--text-dim:      rgba(255, 255, 255, 0.45);
--accent:        rgba(255, 255, 255, 0.95);
--neon:          #ffffff;
```

Also add a status palette (the stock template has no green/amber/red):

```css
--ok:   #4ade80;   /* online  */
--warn: #fbbf24;   /* partial */
--bad:  #f87171;   /* offline / error */
```

> **Why this is the whole fix for "purple everywhere":** `shell.css` only consumes one of
> these vars itself (`body { color: var(--text) }`). All the visible purple came from
> `app.css` *using* `--border`, `--border-active`, `--accent`, etc. So changing the `:root`
> values neutralises the entire app side at once — no need to hunt down individual rules.

### 2. Glass recipes (reuse these everywhere)

**Surface** (panels, cards, modals):
```css
background: rgba(0, 0, 0, 0.38);            /* 0.5–0.68 for modals that need readability */
border: 1px solid var(--border);
border-radius: 14px;                        /* 16–18px for big panels/modals */
backdrop-filter: blur(24px) saturate(1.6);
-webkit-backdrop-filter: blur(24px) saturate(1.6);
```

**Glassy button**:
```css
background: rgba(255, 255, 255, 0.05);
border: 1px solid var(--border);
backdrop-filter: blur(8px);
-webkit-backdrop-filter: blur(8px);
/* hover */
border-color: var(--border-active);
background: rgba(255, 255, 255, 0.1);
color: rgba(255, 255, 255, 0.95);
```

**Modal backdrop** (blur the page behind so the popup reads):
```css
background: rgba(0, 0, 0, 0.55);
backdrop-filter: blur(11px);
-webkit-backdrop-filter: blur(11px);
```

**Hairline divider / row separator**: `rgba(255, 255, 255, 0.08)`.

---

## Rudiments to remove from the template

These are the specific things that made the stock template look unfinished:

1. **Purple `:root` theme** — see §1. The single biggest one.

2. **Flat, non-glassy buttons.** Stock buttons are plain outlines. Give the shared button
   classes the glass recipe (§2): translucent fill + blur, hover lightens. Apply to the
   popover actions (`.pi-action`) and `.login-btn` in `shell.css` so every app inherits it.

3. **`.pi-readme` dim override.** In menu's `app.css` the readme button was overridden to
   `color: var(--text-dim); border-color: var(--border)`, which on the new palette rendered
   as a barely-visible dim chip. Don't dim shared actions — let them use the glassy button
   style. (We replaced it with the glass recipe + a faint tint on logout.)

4. **Inline base64 background image.** The template/app shipped the blurred background as a
   ~57 KB `data:image/webp;base64,…` blob inside `app.css` (bloated the file to ~60 KB and
   can't be cached separately). Use a real file instead:
   - put `background.webp` in `web/`,
   - reference it: `#bg { background-image: url('/background.webp'); }`,
   - register the route (template serves assets as explicit routes — see §"assets").

5. **Purple favicon.** Stock `favicon.svg` uses the old purple (`#c4b5fd`) mark. Swap for a
   neutral dark-glass tile. menu's version: a rounded dark tile (matching the card corner +
   white-12% edge) holding a mini "app list" — rows of `● ▬` with the top dot green
   (`#4ade80`, glowing) to echo the online indicator. Generalise the concept per app.

---

## Assets: every static file needs an explicit route

The template uses a Go 1.22 method-prefix mux and `//go:embed web`, so **each asset must be
registered** as a `GET` route (it won't be auto-served). When adding files:

```go
mux.Handle("GET /background.webp", fileServer)
mux.Handle("GET /app_icons/{file}", fileServer)   // a whole folder
```

Keep this in mind when moving the background out of base64 or adding an icon folder.

---

## Two ways to apply

- **Best — fix the template at the source.** Edit the template's `web/shell.css` `:root`
  (and the shared button classes + favicon). Every app built afterwards is clean with no
  local overrides. This is possible because you own the template.

- **Per-app, when you can't touch `shell.css`** (how menu did it, since `shell.css` is
  read-only there): redefine the same vars in the app's `app.css` `:root`. `app.css` loads
  after `shell.css`, so the override wins. Same values as §1. Use this only as a fallback —
  baking it into the template is cleaner and keeps all apps identical.

---

## Checklist for the template

- [ ] Replace `:root` purple vars with the neutral glass values (§1).
- [ ] Add `--ok` / `--warn` / `--bad` status colours.
- [ ] Apply the glassy-button recipe to `.pi-action` and `.login-btn` (§2).
- [ ] Remove any `.pi-readme`-style dim overrides.
- [ ] Move the background image from inline base64 to `web/background.webp` + route.
- [ ] Replace `favicon.svg` with a neutral dark-glass mark.
- [ ] Confirm `shell.css` still only reads `--text` (so the `:root` swap is the whole job).

Reference implementation: this repo's `build/web/{shell.css overrides in app.css, app.css,
favicon.svg}` and `build/main.go` (asset routes).
