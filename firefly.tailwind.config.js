// firefly.tailwind.config.js
// Serenity UI System — Firefly-inspired design tokens
// Drop this into your tailwind.config.js or merge with your existing config.
//
// Font stack (load via Google Fonts or self-host):
//   Display / Headings : Playfair Display (400, 700)
//   Body               : Crimson Pro (400, 400i, 600)
//   Mono / System      : Share Tech Mono (400)

/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./templates/**/*.templ", "./static/**/*.js"],
  theme: {
    extend: {

      // ── COLOR PALETTE ─────────────────────────────────────────────────────
      colors: {
        // Warm foreground — ship interior, lantern light, dust
        rust: {
          DEFAULT:  "#C4531A",
          light:    "#E8784A",
          dark:     "#8A3610",
          dim:      "rgba(196,83,26,0.15)",
          border:   "rgba(196,83,26,0.25)",
        },
        amber: {
          DEFAULT:  "#D4922A",
          light:    "#F0B84A",
          dark:     "#9A6A1A",
          dim:      "rgba(212,146,42,0.12)",
        },
        bone:      "#E8DCC8",
        parchment: {
          DEFAULT:  "#D4C5A0",
          dark:     "#B8A882",
        },
        gold: {
          DEFAULT:  "#C8AA50",
          light:    "#E0C870",
        },

        // Cool background — the Black, deep space
        slate: {
          DEFAULT:  "#3A4A5C",
          light:    "#4E6278",
          dark:     "#232F3C",
          deep:     "#151E28",
        },
        space: {
          DEFAULT:  "#0D1520",
          void:     "#070D15",
        },

        // Accent — Cortex system, jade terminal green
        jade: {
          DEFAULT:  "#3D7A6A",
          light:    "#5BA090",
          dark:     "#274F44",
        },

        // Semantic
        alliance: "#B83030",   // danger / error / enemy
        cortex:   "#8BAAC4",   // info / system message
      },

      // ── FONTS ──────────────────────────────────────────────────────────────
      fontFamily: {
        display: ['"Playfair Display"', "Georgia", "serif"],
        body:    ['"Crimson Pro"',      "Georgia", "serif"],
        mono:    ['"Share Tech Mono"',  "monospace"],
        // Fallback sans for utility text
        sans:    ["ui-sans-serif", "system-ui", "sans-serif"],
      },

      // ── FONT SIZES ─────────────────────────────────────────────────────────
      fontSize: {
        "2xs":   ["10px", { lineHeight: "1.4", letterSpacing: "0.2em" }],
        xs:      ["12px", { lineHeight: "1.5" }],
        sm:      ["13px", { lineHeight: "1.6" }],
        base:    ["16px", { lineHeight: "1.7" }],
        lg:      ["18px", { lineHeight: "1.6" }],
        xl:      ["22px", { lineHeight: "1.4" }],
        "2xl":   ["28px", { lineHeight: "1.2" }],
        "3xl":   ["36px", { lineHeight: "1.1" }],
        "4xl":   ["48px", { lineHeight: "1.0" }],
      },

      // ── LETTER SPACING ─────────────────────────────────────────────────────
      letterSpacing: {
        tight:    "-0.01em",
        normal:    "0em",
        wide:      "0.06em",
        wider:     "0.12em",
        widest:    "0.25em",
        "cortex":  "0.3em",    // mono labels / terminal output
      },

      // ── BORDER RADIUS ──────────────────────────────────────────────────────
      borderRadius: {
        none:  "0",
        sm:    "2px",
        DEFAULT:"3px",
        md:    "4px",
        lg:    "6px",
        xl:    "8px",
        full:  "9999px",
      },

      // ── BORDER WIDTH ───────────────────────────────────────────────────────
      borderWidth: {
        DEFAULT: "1px",
        0:       "0",
        half:    "0.5px",
        2:       "2px",
        3:       "3px",          // left-border accent on cards
      },

      // ── SHADOWS ────────────────────────────────────────────────────────────
      // Minimal — glow-only, no drop shadows
      boxShadow: {
        none:    "none",
        rust:    "0 0 0 2px rgba(196,83,26,0.4)",
        amber:   "0 0 0 2px rgba(212,146,42,0.4)",
        jade:    "0 0 0 2px rgba(61,122,106,0.4)",
        focus:   "0 0 0 2px rgba(196,83,26,0.6)",  // focus ring
      },

      // ── BACKGROUND OPACITY VARIANTS ────────────────────────────────────────
      // Use with bg-rust/15, bg-jade/20 etc. via Tailwind opacity modifier.
      // No custom tokens needed — covered by color + opacity syntax.

      // ── SPACING EXTRAS ─────────────────────────────────────────────────────
      spacing: {
        "4.5": "18px",
        "18":  "72px",
        "22":  "88px",
        "30":  "120px",
      },

      // ── TRANSITION TIMING ──────────────────────────────────────────────────
      transitionTimingFunction: {
        DEFAULT: "cubic-bezier(0.4, 0, 0.2, 1)",
      },
      transitionDuration: {
        DEFAULT: "150ms",
      },
    },
  },

  plugins: [

    // ── UTILITY PLUGIN ──────────────────────────────────────────────────────
    // Registers semantic component classes that pair well with the palette.
    // Usage (in templ): class="btn-primary", class="card-vessel", etc.
    function ({ addComponents, addUtilities, theme }) {

      addComponents({

        // ── BUTTONS ─────────────────────────────────────────────────────────
        ".btn": {
          display:        "inline-flex",
          alignItems:     "center",
          gap:            "6px",
          fontFamily:     theme("fontFamily.mono"),
          fontSize:       "11px",
          letterSpacing:  "0.2em",
          textTransform:  "uppercase",
          padding:        "10px 20px",
          borderRadius:   theme("borderRadius.DEFAULT"),
          cursor:         "pointer",
          border:         "1px solid transparent",
          transition:     "all 150ms",
          userSelect:     "none",
          textDecoration: "none",
        },
        ".btn-primary": {
          background:     theme("colors.rust.DEFAULT"),
          color:          theme("colors.bone"),
          borderColor:    theme("colors.rust.light"),
          "&:hover":      { background: theme("colors.rust.light") },
        },
        ".btn-secondary": {
          background:     "transparent",
          color:          theme("colors.parchment.DEFAULT"),
          borderColor:    theme("colors.rust.border"),
          "&:hover":      {
            borderColor:  theme("colors.rust.DEFAULT"),
            color:        theme("colors.bone"),
          },
        },
        ".btn-ghost": {
          background:     "transparent",
          color:          theme("colors.amber.light"),
          "&:hover":      { color: theme("colors.amber.DEFAULT") },
        },
        ".btn-danger": {
          background:     "transparent",
          color:          theme("colors.alliance"),
          borderColor:    theme("colors.alliance"),
          "&:hover":      { background: "rgba(184,48,48,0.12)" },
        },

        // ── CARDS ────────────────────────────────────────────────────────────
        ".card-vessel": {
          background:     theme("colors.slate.dark"),
          border:         "1px solid rgba(196,83,26,0.25)",
          borderRadius:   theme("borderRadius.md"),
          padding:        "20px",
          position:       "relative",
          "&::before": {
            content:      '""',
            position:     "absolute",
            top:          "0",
            left:         "0",
            width:        "3px",
            height:       "100%",
            background:   theme("colors.rust.DEFAULT"),
            borderRadius: "4px 0 0 4px",
          },
        },
        ".card-cortex": {
          background:     theme("colors.space.void"),
          border:         "1px solid rgba(61,122,106,0.3)",
          borderRadius:   theme("borderRadius.md"),
          padding:        "16px 20px",
          fontFamily:     theme("fontFamily.mono"),
          fontSize:       "12px",
          color:          theme("colors.jade.light"),
          lineHeight:     "2",
        },

        // ── FORM INPUTS ──────────────────────────────────────────────────────
        ".input-cortex": {
          background:     theme("colors.slate.dark"),
          border:         "1px solid rgba(196,83,26,0.25)",
          borderRadius:   theme("borderRadius.DEFAULT"),
          padding:        "10px 14px",
          color:          theme("colors.bone"),
          fontFamily:     theme("fontFamily.body"),
          fontSize:       "15px",
          width:          "100%",
          outline:        "none",
          transition:     "border-color 150ms",
          "&::placeholder": { color: theme("colors.parchment.dark") },
          "&:focus":      {
            borderColor:  theme("colors.rust.DEFAULT"),
            boxShadow:    theme("boxShadow.focus"),
          },
        },

        // ── BADGES ───────────────────────────────────────────────────────────
        ".badge": {
          fontFamily:     theme("fontFamily.mono"),
          fontSize:       "10px",
          letterSpacing:  "0.2em",
          textTransform:  "uppercase",
          padding:        "3px 10px",
          borderRadius:   theme("borderRadius.sm"),
          display:        "inline-block",
        },
        ".badge-crew":    { background: "rgba(196,83,26,0.2)",  color: theme("colors.rust.light"),  border: "1px solid rgba(196,83,26,0.4)" },
        ".badge-guild":   { background: "rgba(61,122,106,0.2)", color: theme("colors.jade.light"),  border: "1px solid rgba(61,122,106,0.4)" },
        ".badge-cortex":  { background: "rgba(78,98,120,0.2)",  color: theme("colors.cortex"),      border: "1px solid rgba(78,98,120,0.4)" },
        ".badge-wanted":  { background: "rgba(184,48,48,0.2)",  color: "#D47070",                   border: "1px solid rgba(184,48,48,0.4)" },
        ".badge-gold":    { background: "rgba(200,170,80,0.2)", color: theme("colors.gold.DEFAULT"),border: "1px solid rgba(200,170,80,0.4)" },

        // ── ALERTS ───────────────────────────────────────────────────────────
        ".alert": {
          borderRadius:   theme("borderRadius.DEFAULT"),
          padding:        "12px 16px",
          display:        "flex",
          gap:            "12px",
          alignItems:     "flex-start",
          borderLeft:     "3px solid",
          fontSize:       "14px",
          lineHeight:     "1.5",
        },
        ".alert-warning": { background: "rgba(212,146,42,0.1)",  borderColor: theme("colors.amber.DEFAULT"),   color: theme("colors.amber.light") },
        ".alert-danger":  { background: "rgba(184,48,48,0.1)",   borderColor: theme("colors.alliance"),        color: "#D47070" },
        ".alert-info":    { background: "rgba(78,98,120,0.2)",   borderColor: theme("colors.cortex"),          color: theme("colors.cortex") },
        ".alert-success": { background: "rgba(61,122,106,0.1)",  borderColor: theme("colors.jade.DEFAULT"),    color: theme("colors.jade.light") },

        // ── SECTION LABEL ─────────────────────────────────────────────────────
        // Usage: <div class="section-label">Color Palette</div>
        ".section-label": {
          fontFamily:     theme("fontFamily.mono"),
          fontSize:       "10px",
          letterSpacing:  "0.3em",
          textTransform:  "uppercase",
          color:          theme("colors.rust.DEFAULT"),
          display:        "flex",
          alignItems:     "center",
          gap:            "12px",
          marginBottom:   "28px",
          "&::after": {
            content:      '""',
            flex:         "1",
            height:       "1px",
            background:   "rgba(196,83,26,0.25)",
            maxWidth:     "200px",
          },
        },

        // ── METER / PROGRESS BAR ──────────────────────────────────────────────
        ".meter-track": {
          height:         "4px",
          background:     theme("colors.slate.dark"),
          borderRadius:   "2px",
          overflow:       "hidden",
        },
        ".meter-fill": {
          height:         "100%",
          borderRadius:   "2px",
          transition:     "width 400ms ease",
        },
        ".meter-rust":    { background: theme("colors.rust.DEFAULT") },
        ".meter-amber":   { background: theme("colors.amber.DEFAULT") },
        ".meter-jade":    { background: theme("colors.jade.DEFAULT") },
        ".meter-red":     { background: theme("colors.alliance") },
      });

      addUtilities({
        // Cortex scanline overlay (apply to positioned wrapper)
        ".scanlines": {
          position:   "relative",
          "&::after": {
            content:         '""',
            position:        "absolute",
            inset:           "0",
            backgroundImage: "repeating-linear-gradient(0deg, transparent, transparent 2px, rgba(0,0,0,0.08) 2px, rgba(0,0,0,0.08) 4px)",
            pointerEvents:   "none",
            zIndex:          "10",
          },
        },
        // Mono label utility
        ".label-mono": {
          fontFamily:    '"Share Tech Mono", monospace',
          fontSize:      "10px",
          letterSpacing: "0.25em",
          textTransform: "uppercase",
          color:         theme("colors.parchment.dark"),
        },
        // Amber glow on text
        ".text-glow-amber": {
          textShadow: "0 0 12px rgba(212,146,42,0.5)",
        },
        ".text-glow-jade": {
          textShadow: "0 0 12px rgba(61,122,106,0.6)",
        },
      });
    },
  ],
};
