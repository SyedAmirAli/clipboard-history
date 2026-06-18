# clipd — Marketing Landing Page · Build Brief

> A complete, self‑contained brief for building **one** polished, single‑file,
> multi‑section **marketing landing page** for **clipd** (a native clipboard‑history app).
> Hand this whole file to Claude Code / Claude cowork and ask it to produce `index.html`.
> Goal: **drive GitHub stars + downloads** and make the project look professional.

---

## 0. Objective & audience

- **Product:** *clipd* — a fast, native clipboard history manager for **Windows** and **Linux**. Text + image history, global hotkey, search, pinning, an encrypted private vault, save‑to‑file/export, system tray, light/dark themes.
- **Audience:** developers and power users on Windows/Linux who copy‑paste constantly.
- **Primary goals (in priority order):**
  1. Get the visitor to **⭐ star the GitHub repo**.
  2. Get them to **download** the right build (Windows `.exe` / Linux `.deb`).
  3. Make the author look credible (portfolio / contact).
- **Tone:** modern, confident, "intelligent dev tool." Clean, generous whitespace, subtle motion. Think Linear / Raycast / Vercel landing pages.

---

## 1. Hard technical constraints (must follow exactly)

1. **Single static `index.html`** — no build step, no framework, no bundler. Works by double‑clicking the file or any static host (GitHub Pages, Netlify, Vercel).
2. **Tailwind CSS v4 via CDN — use this exact tag** in `<head>`:
   ```html
   <script src="https://cdn.jsdelivr.net/npm/@tailwindcss/browser@4"></script>
   ```
3. **Dual theme (light + dark)** with a **manual toggle** *and* respect for the OS preference, persisted in `localStorage`, applied **before first paint** (no flash). Implementation details in §6.
4. **Attractive, "intelligent application" typography** via Google Fonts (see §2): a geometric display face for headings + a clean grotesk for body + a mono face for keyboard keys/code.
5. **Fully responsive** (mobile‑first), accessible (semantic landmarks, alt text, focus states, `aria-*`, keyboard‑operable toggles), and fast (lazy‑load gallery images).
6. Only vanilla JS (small, inline) for: theme toggle, mobile nav, gallery light/dark switch, smooth scroll, copy‑to‑clipboard on the install command, and reveal‑on‑scroll.

---

## 2. Brand & design system

### Typography (Google Fonts — put in `<head>`)
```html
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@500;600;700&family=Inter:wght@400;500;600&family=JetBrains+Mono:wght@500&display=swap" rel="stylesheet">
```
- **Display / headings:** `Space Grotesk` (600/700) — techy, geometric, confident.
- **Body / UI:** `Inter` (400/500/600) — crisp, neutral.
- **Mono (keyboard keys, commands):** `JetBrains Mono` (500).

Wire them into Tailwind v4 with a theme block (see skeleton §7) so you can use `font-display`, `font-sans`, `font-mono`.

### Color & vibe
- **Accent:** an electric **violet → indigo** gradient (e.g. `#7c3aed → #4f46e5`) for primary CTAs, links, highlights. Optional secondary accent: cyan `#06b6d4` for small details.
- **Light theme:** near‑white background (`#fafafa`/`#ffffff`), slate text (`#0f172a`), soft gray borders.
- **Dark theme:** deep slate/near‑black (`#0b0b12` / `#0f172a`), light slate text (`#e2e8f0`), subtle violet glow accents.
- **Surfaces:** rounded‑2xl cards, 1px subtle borders, soft shadows, faint gradient/grid or "aurora" blur blobs behind the hero.
- **Radius:** generous (`rounded-2xl`/`rounded-3xl`). **Shadows:** soft, layered. **Motion:** 150–300ms ease, reveal‑on‑scroll, gentle hover lifts. Respect `prefers-reduced-motion`.

---

## 3. Page structure (sections, in order)

1. **Sticky nav** — logo "clipd", anchor links (Features · Screenshots · Download · About), **theme toggle**, **GitHub ⭐ Star** button, primary **Download** button. Collapses to a hamburger on mobile.
2. **Hero** — big headline + subhead, two primary CTAs (**Download for Windows**, **Star on GitHub**) + a secondary Linux link, trust line ("Free & open source · ~15 MB · no installer"), and a **hero screenshot/mockup** (use a strong light or dark shot; ideally swaps with the theme).
3. **Star banner** — a slim, friendly call‑to‑action strip: "If clipd saves you time, please ⭐ the repo — it really helps." with a star button + live‑looking star badge (use the GitHub shields badge img).
4. **Features grid** — 6–9 cards (see §4 for the exact list/copy), each with an icon (inline SVG), title, one‑line description.
5. **Screenshots gallery** — tabbed or toggle **Light / Dark**, responsive grid/carousel of the app screenshots (§5). The gallery's light/dark choice should default to the current page theme.
6. **How it works / Quick start** — 3 steps (Download → Run → press **Win + J**), plus the copy‑able CLI line.
7. **Download** — two big cards: **Windows (.exe)** and **Linux (.deb)** with version + the exact URLs (§8), file size, and "no install needed" notes. Mention the GitHub Releases page too.
8. **Built with / Open source** — small strip: Go · Wails v2 · TypeScript · pure‑Go SQLite, single ~15 MB binary; link to the repo and explain the branch layout (`windows` = Windows build, `master` = Linux/cross‑platform).
9. **About the developer** — short bio, avatar/initials, links to portfolio, LinkedIn, email, phone; a CTA to "follow / star."
10. **Footer** — repo link, download links, contact, copyright "© 2026 Syed Amir Ali", "Built with clipd in mind" + one more ⭐ Star button.

---

## 4. Section content & copy (use this; refine wording freely)

### Hero
- **Eyebrow:** `Native · Windows & Linux · Open source`
- **Headline (display font):** **"Your clipboard, with a memory."**
- **Subhead:** "clipd quietly remembers everything you copy — text and images — so you can search it, pin it, paste it, and never lose a snippet again. Lightning‑fast, native, and private."
- **CTAs:** `⬇ Download for Windows` (primary, → Windows URL) · `⭐ Star on GitHub` (secondary, → repo). Under them, a small link: `Linux (.deb) →`.
- **Trust line:** "Free & open source · single ~15 MB binary · no installer · works offline."

### Features (cards — title + one‑liner)
1. **Text & image history** — Every copy is saved with thumbnails; scroll back through your whole clipboard.
2. **Instant search & filters** — Live search with All / Text / Images / Pinned filters.
3. **Global hotkey — Win + J** — Summons the popup **at your cursor**, on the right monitor, instantly.
4. **Pin what matters** — Pin snippets so they never get evicted from history.
5. **Private Vault** — PIN/password‑protected, encrypted storage for sensitive clips.
6. **Save to file & Export ZIP** — Save any item (sorted into `text/` & `images/`) or export your whole history to a ZIP.
7. **System tray + taskbar** — Lives in the tray with a full menu; optional taskbar button; auto‑start on login.
8. **Light · Dark · Auto** — A beautiful frameless UI that matches your system theme.
9. **Native & fast** — Built on the Win32 clipboard API (no heavy deps); reliable image/screenshot capture.

### How it works (3 steps)
1. **Download** the single `.exe` (Windows) or `.deb` (Linux).
2. **Run it** — no install, no admin. It starts in the tray.
3. **Press `Win + J`** to open your history anywhere. Type to search, Enter to paste.

CLI line (make it copy‑to‑clipboard with a button):
```bash
clipd toggle   # show/hide the popup · clipd quit to exit
```

### Download cards
- **Windows 10 / 11 (x64)** — `clipd‑v2.0.0.exe` · ~15 MB · portable, no installer. Button → Windows URL (§8).
- **Linux — Ubuntu / Mint (amd64)** — `clipd_1.1.5_amd64.deb`. Button → Linux URL (§8). Note: `sudo apt install ./clipd_1.1.5_amd64.deb`.
- Secondary: "See all releases on GitHub →" (→ repo `/releases`).

### About the developer
- **Name:** Syed Amir Ali
- **Blurb:** "Full‑stack developer. I build fast, native, thoughtful tools. clipd is one of them — if it helps you, a ⭐ on GitHub means a lot."
- **Links:** Portfolio · LinkedIn · Email · Phone (§8).

---

## 5. Screenshot assets

**Source folder (copy these into the project):**
`C:\Users\syeda\Downloads\Clipd\clipd-export-2026-06-19\images`

Copy them into `./assets/screenshots/` and reference as `assets/screenshots/<name>`.

- **Dark theme (8):** `dark-1.png` … `dark-8.png`
- **Light theme (11):** `light-1.png` … `light-11.png`

**Usage guidance:**
- **Hero image:** pick one strong, representative shot (e.g. `dark-1.png` for the default‑dark hero, with `light-1.png` as the light‑theme swap).
- **Gallery:** show the **dark** set when the page is in dark mode and the **light** set in light mode (and/or a Light/Dark tab the user can flip). Lazy‑load (`loading="lazy"`), `decoding="async"`, descriptive `alt` (e.g. "clipd history list with image thumbnails — dark theme"), rounded corners + border + soft shadow, click‑to‑zoom (lightbox) is a nice‑to‑have.
- Keep aspect ratios intact; constrain with `max-w` and `object-contain`. Add a subtle browser/window frame ("mockup") around hero/gallery images for polish.

---

## 6. Theme toggle (Tailwind v4 + class strategy)

Tailwind v4's `dark:` variant defaults to the OS preference. To support a **manual toggle**, switch it to a **class strategy** by adding this custom variant in the Tailwind config block (see skeleton), then toggle a `.dark` class on `<html>`.

**Anti‑flash script (place in `<head>` BEFORE the body renders):**
```html
<script>
  (function () {
    const stored = localStorage.getItem('theme');
    const dark = stored ? stored === 'dark'
      : matchMedia('(prefers-color-scheme: dark)').matches;
    document.documentElement.classList.toggle('dark', dark);
  })();
</script>
```
**Toggle handler (defer/end of body):** flips `.dark`, writes `localStorage.theme`, updates the toggle icon (sun/moon) and `aria-pressed`, and re‑points the gallery to the matching screenshot set.

---

## 7. Starter skeleton (give the implementer this exact head/setup)

```html
<!doctype html>
<html lang="en" class="scroll-smooth">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>clipd — clipboard history with a memory · Windows & Linux</title>
  <meta name="description" content="clipd is a fast, native, open‑source clipboard history manager for Windows and Linux. Text & image history, global hotkey, search, pinning, encrypted vault, dark/light themes." />
  <meta property="og:title" content="clipd — your clipboard, with a memory" />
  <meta property="og:description" content="Fast, native, open‑source clipboard history for Windows & Linux." />
  <meta property="og:type" content="website" />
  <!-- og:image: a 1200x630 social card screenshot if available -->

  <!-- Anti-flash theme (must run before paint) -->
  <script>
    (function () {
      const s = localStorage.getItem('theme');
      const d = s ? s === 'dark' : matchMedia('(prefers-color-scheme: dark)').matches;
      document.documentElement.classList.toggle('dark', d);
    })();
  </script>

  <!-- Fonts -->
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Space+Grotesk:wght@500;600;700&family=Inter:wght@400;500;600&family=JetBrains+Mono:wght@500&display=swap" rel="stylesheet">

  <!-- Tailwind v4 (REQUIRED CDN) -->
  <script src="https://cdn.jsdelivr.net/npm/@tailwindcss/browser@4"></script>

  <!-- Tailwind v4 config: class-based dark + fonts + accent -->
  <style type="text/tailwindcss">
    @custom-variant dark (&:where(.dark, .dark *));
    @theme {
      --font-display: "Space Grotesk", ui-sans-serif, system-ui, sans-serif;
      --font-sans: "Inter", ui-sans-serif, system-ui, sans-serif;
      --font-mono: "JetBrains Mono", ui-monospace, monospace;
      --color-brand-500: #7c3aed;
      --color-brand-600: #6d28d9;
      --color-brand-700: #5b21b6;
    }
  </style>
</head>
<body class="font-sans antialiased bg-white text-slate-900 dark:bg-[#0b0b12] dark:text-slate-100 selection:bg-brand-500/30">
  <!-- nav · hero · star-banner · features · screenshots · how-it-works · download · built-with · about · footer -->
  <!-- Build all sections from §3 / §4. Headings use class="font-display". -->
</body>
</html>
```

Notes for the implementer:
- Use `class="font-display"` on headings, `font-mono` on `<kbd>`/commands.
- Primary buttons: brand gradient (`bg-gradient-to-r from-brand-500 to-indigo-600`), white text, `rounded-xl`, hover lift + ring.
- Render keyboard shortcuts as styled `<kbd>` chips: `Win` `+` `J`.
- Hero background: layered radial/aurora blur blobs in brand colors at low opacity, plus an optional faint grid; all behind content, `pointer-events-none`.

---

## 8. Content reference (exact values — do not change)

**Downloads**
- **Windows (.exe):** `https://github.com/SyedAmirAli/clipboard-history/releases/download/v2.0.0/clipd-v2.0.0.exe`
- **Linux (.deb, Ubuntu/Mint amd64):** `https://github.com/SyedAmirAli/clipboard-history/releases/download/v1.1.5/clipd_1.1.5_amd64.deb`

**Repository**
- **Main repo:** `https://github.com/SyedAmirAli/clipboard-history`
  - Branch **`windows`** → native Windows build · Branch **`master`** → Linux / cross‑platform.
  - Star button → repo; "all releases" → `https://github.com/SyedAmirAli/clipboard-history/releases`.
- **GitHub star badge (optional, for the star banner):**
  `https://img.shields.io/github/stars/SyedAmirAli/clipboard-history?style=social`

**Developer**
- **Portfolio:** `https://portfolio.syedamirali.me`
- **LinkedIn:** `https://www.linkedin.com/in/syedamirali473`
- **Email:** `syedamirali473@gmail.com`
- **Phone:** `+880 17807594`
- **Name / copyright:** Syed Amir Ali · © 2026

**App facts (for copy)**
- Name: **clipd** · Version: **v2.0.0** (Windows) / **v1.1.5** (Linux `.deb`).
- Built with **Go + Wails v2 + TypeScript + pure‑Go SQLite**; single **~15 MB** native binary.
- Key features: text+image history, global hotkey **Win + J** (opens at cursor, multi‑monitor aware), live search + filters, pinning, **Private Vault** (encrypted), **Save to file** (`text/`+`images/` subfolders) and **Export all to ZIP**, system tray menu + taskbar button, auto‑start, light/dark/auto themes.

---

## 9. Accessibility · responsiveness · SEO

- Semantic landmarks (`<header> <nav> <main> <section> <footer>`), one `<h1>` (hero), logical heading order.
- All images have meaningful `alt`. Decorative blobs are `aria-hidden`.
- Theme toggle and mobile menu are real `<button>`s, keyboard‑operable, with `aria-label`/`aria-pressed`/`aria-expanded`.
- Visible focus rings; color contrast ≥ WCAG AA in **both** themes.
- Respect `prefers-reduced-motion` (disable reveal/parallax).
- Mobile‑first; verify at 360px, 768px, 1280px. Gallery becomes a swipeable/stacked layout on mobile.
- Meta title/description + Open Graph (skeleton has them); optional `theme-color` per scheme.

---

## 10. Acceptance checklist (definition of done)

- [ ] Single `index.html`, opens by double‑click, no console errors.
- [ ] Uses the **exact** Tailwind v4 CDN tag from §1.
- [ ] Light + dark themes both look great; toggle persists; **no flash** on load; respects OS default.
- [ ] All 10 sections from §3 present and responsive (360 / 768 / 1280).
- [ ] Fonts loaded (Space Grotesk / Inter / JetBrains Mono).
- [ ] Hero + gallery use the real screenshots; light set in light mode, dark set in dark mode; images lazy‑load with alt text.
- [ ] Every URL/contact from §8 is correct and links open in a new tab (`rel="noopener"`).
- [ ] At least **3** prominent "⭐ Star on GitHub" CTAs (nav, star banner, footer).
- [ ] Download buttons point to the exact `.exe` / `.deb` URLs.
- [ ] Copy‑to‑clipboard works on the CLI line; smooth‑scroll anchors work.
- [ ] Accessible (landmarks, focus, contrast, reduced‑motion) and SEO meta present.
```
