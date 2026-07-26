// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

/**
 * Self-built hero schematic: Flux reconcile → bridge observe → GitHub Deployment.
 * Pure SVG + CSS motion (no external assets).
 */
export function HeroBridgeGraphic() {
  return (
    <div className="gdb-hero-graphic" aria-hidden="true">
      <svg
        className="gdb-hero-graphic__svg"
        viewBox="0 0 560 420"
        role="img"
        xmlns="http://www.w3.org/2000/svg"
      >
        <title>Flux to GitHub Deployments via the bridge</title>
        <defs>
          <linearGradient id="gdb-panel" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="var(--gdb-paper-elevated)" />
            <stop offset="100%" stopColor="var(--gdb-paper)" />
          </linearGradient>
          <linearGradient id="gdb-beam" x1="0" y1="0" x2="1" y2="0">
            <stop offset="0%" stopColor="var(--gdb-accent)" stopOpacity="0.15" />
            <stop offset="50%" stopColor="var(--gdb-accent)" stopOpacity="0.55" />
            <stop offset="100%" stopColor="var(--gdb-accent)" stopOpacity="0.15" />
          </linearGradient>
          <filter id="gdb-soft" x="-20%" y="-20%" width="140%" height="140%">
            <feDropShadow
              dx="0"
              dy="10"
              stdDeviation="14"
              floodColor="var(--gdb-ink)"
              floodOpacity="0.08"
            />
          </filter>
        </defs>

        {/* Stage card */}
        <rect
          className="gdb-hero-graphic__stage"
          x="16"
          y="28"
          width="528"
          height="364"
          rx="18"
          fill="url(#gdb-panel)"
          stroke="var(--gdb-line)"
          strokeWidth="1.25"
          filter="url(#gdb-soft)"
        />

        {/* Fine grid */}
        <g className="gdb-hero-graphic__grid" opacity="0.55">
          {Array.from({ length: 10 }, (_, i) => (
            <line
              key={`v-${i}`}
              x1={56 + i * 48}
              y1="48"
              x2={56 + i * 48}
              y2="372"
              stroke="var(--gdb-line)"
              strokeWidth="1"
            />
          ))}
          {Array.from({ length: 7 }, (_, i) => (
            <line
              key={`h-${i}`}
              x1="36"
              y1={68 + i * 48}
              x2="524"
              y2={68 + i * 48}
              stroke="var(--gdb-line)"
              strokeWidth="1"
            />
          ))}
        </g>

        {/* Header strip */}
          <text
            x="40"
            y="64"
            className="gdb-hero-graphic__mono"
            fill="var(--gdb-accent-strong)"
            fontSize="13"
            letterSpacing="0.08em"
          >
            RECONCILE → REPORT
          </text>
          <text
            x="520"
            y="64"
            textAnchor="end"
            className="gdb-hero-graphic__mono"
            fill="var(--rp-c-text-3)"
            fontSize="13"
          >
            observe only
          </text>

        {/* Left: Flux */}
        <g className="gdb-hero-graphic__node gdb-hero-graphic__node--flux">
          <rect
            x="40"
            y="108"
            width="148"
            height="168"
            rx="12"
            fill="var(--gdb-paper-elevated)"
            stroke="var(--rp-c-divider)"
            strokeWidth="1.25"
          />
          <circle cx="60" cy="130" r="4" fill="var(--gdb-accent)" />
          <text
            x="74"
            y="135"
            className="gdb-hero-graphic__label"
            fill="var(--gdb-ink)"
            fontSize="15"
            fontWeight="600"
          >
            Flux
          </text>
          <rect
            x="56"
            y="158"
            width="116"
            height="36"
            rx="8"
            fill="var(--gdb-accent-soft)"
            stroke="var(--gdb-accent)"
            strokeWidth="1"
            strokeOpacity="0.35"
          />
          <text
            x="114"
            y="181"
            textAnchor="middle"
            className="gdb-hero-graphic__mono"
            fill="var(--gdb-accent-strong)"
            fontSize="12"
          >
            Kustomization
          </text>
          <rect
            x="56"
            y="206"
            width="116"
            height="36"
            rx="8"
            fill="var(--gdb-accent-soft)"
            stroke="var(--gdb-accent)"
            strokeWidth="1"
            strokeOpacity="0.35"
          />
          <text
            x="114"
            y="229"
            textAnchor="middle"
            className="gdb-hero-graphic__mono"
            fill="var(--gdb-accent-strong)"
            fontSize="12"
          >
            HelmRelease
          </text>
          <text
            x="114"
            y="260"
            textAnchor="middle"
            className="gdb-hero-graphic__mono"
            fill="var(--rp-c-text-3)"
            fontSize="11"
          >
            Ready · Reconciling
          </text>
        </g>

        {/* Bridge beam + traveling packets */}
        <g className="gdb-hero-graphic__bridge">
          <path
            d="M188 192 H246"
            stroke="url(#gdb-beam)"
            strokeWidth="6"
            strokeLinecap="round"
          />
          <path
            className="gdb-hero-graphic__rail"
            d="M188 192 H246"
            fill="none"
            stroke="var(--gdb-accent)"
            strokeWidth="1.5"
            strokeDasharray="4 8"
            strokeLinecap="round"
          />
          <circle className="gdb-hero-graphic__packet gdb-hero-graphic__packet--a" r="4.5" fill="var(--gdb-accent)">
            <animateMotion dur="2.4s" repeatCount="indefinite" path="M188 192 H246" />
          </circle>

          {/* Center bridge hub */}
          <g transform="translate(280 192)">
            <circle
              r="42"
              fill="var(--gdb-paper-elevated)"
              stroke="var(--gdb-accent)"
              strokeWidth="1.5"
              strokeOpacity="0.45"
            />
            <circle
              className="gdb-hero-graphic__ring"
              r="42"
              fill="none"
              stroke="var(--gdb-accent)"
              strokeWidth="1.25"
              strokeDasharray="6 10"
              strokeOpacity="0.7"
            />
            <path
              d="M-18 0h8M10 0h8"
              stroke="var(--gdb-accent)"
              strokeWidth="3"
              strokeLinecap="round"
            />
            <circle cx="-6" cy="0" r="7" fill="none" stroke="var(--gdb-ink)" strokeWidth="2.25" />
            <circle cx="6" cy="0" r="7" fill="none" stroke="var(--gdb-ink)" strokeWidth="2.25" />
            <text
              y="70"
              textAnchor="middle"
              className="gdb-hero-graphic__label"
              fill="var(--gdb-ink)"
              fontSize="15"
              fontWeight="600"
            >
              Bridge
            </text>
            <text
              y="88"
              textAnchor="middle"
              className="gdb-hero-graphic__mono"
              fill="var(--rp-c-text-3)"
              fontSize="11"
            >
              OCI · annotations
            </text>
          </g>

          <path
            d="M322 192 H372"
            stroke="url(#gdb-beam)"
            strokeWidth="6"
            strokeLinecap="round"
          />
          <path
            className="gdb-hero-graphic__rail"
            d="M322 192 H372"
            fill="none"
            stroke="var(--gdb-accent)"
            strokeWidth="1.5"
            strokeDasharray="4 8"
            strokeLinecap="round"
          />
          <circle className="gdb-hero-graphic__packet gdb-hero-graphic__packet--b" r="4.5" fill="var(--gdb-accent)">
            <animateMotion dur="2.4s" begin="0.7s" repeatCount="indefinite" path="M322 192 H372" />
          </circle>
        </g>

        {/* Right: GitHub Deployments */}
        <g className="gdb-hero-graphic__node gdb-hero-graphic__node--gh">
          <rect
            x="372"
            y="108"
            width="148"
            height="168"
            rx="12"
            fill="var(--gdb-paper-elevated)"
            stroke="var(--rp-c-divider)"
            strokeWidth="1.25"
          />
          <circle cx="392" cy="130" r="4" fill="var(--gdb-ink)" />
          <text
            x="406"
            y="135"
            className="gdb-hero-graphic__label"
            fill="var(--gdb-ink)"
            fontSize="15"
            fontWeight="600"
          >
            GitHub
          </text>
          <text
            x="446"
            y="158"
            textAnchor="middle"
            className="gdb-hero-graphic__mono"
            fill="var(--rp-c-text-3)"
            fontSize="11"
          >
            Deployments API
          </text>

          {/* Status ladder */}
          <g className="gdb-hero-graphic__statuses">
            <g className="gdb-hero-graphic__status gdb-hero-graphic__status--queued">
              <circle cx="396" cy="186" r="5.5" fill="var(--gdb-accent-soft)" stroke="var(--gdb-accent)" strokeWidth="1.5" />
              <text x="412" y="191" className="gdb-hero-graphic__mono" fill="var(--rp-c-text-2)" fontSize="13">
                queued
              </text>
            </g>
            <g className="gdb-hero-graphic__status gdb-hero-graphic__status--progress">
              <circle cx="396" cy="218" r="5.5" fill="var(--gdb-accent-soft)" stroke="var(--gdb-accent)" strokeWidth="1.5" />
              <text x="412" y="223" className="gdb-hero-graphic__mono" fill="var(--rp-c-text-2)" fontSize="13">
                in_progress
              </text>
            </g>
            <g className="gdb-hero-graphic__status gdb-hero-graphic__status--success">
              <circle cx="396" cy="250" r="5.5" fill="var(--gdb-accent)" />
              <text x="412" y="255" className="gdb-hero-graphic__mono" fill="var(--gdb-ink)" fontSize="13" fontWeight="500">
                success
              </text>
            </g>
          </g>
        </g>

        {/* Bottom ticker */}
        <g className="gdb-hero-graphic__ticker">
          <rect
            x="40"
            y="304"
            width="480"
            height="64"
            rx="12"
            fill="var(--gdb-ink)"
          />
          <text
            x="60"
            y="332"
            className="gdb-hero-graphic__mono"
            fill="var(--gdb-accent)"
            fontSize="11"
            letterSpacing="0.06em"
          >
            DEPLOYMENT
          </text>
          <text
            x="60"
            y="352"
            className="gdb-hero-graphic__mono"
            fill="#e8ebe8"
            fontSize="13"
          >
            owner/repo · production · abc1234
          </text>
        </g>
      </svg>
    </div>
  );
}
