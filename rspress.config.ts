// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

import * as path from "node:path";
import { defineConfig, type UserConfig } from "@rspress/core";
import mermaid from "rspress-plugin-mermaid";

const isGitHubActions = process.env.GITHUB_ACTIONS === "true";

const config: UserConfig = {
  root: path.join(__dirname, "docs"),
  title: "GitHub Deployment Bridge",
  description:
    "Lightweight Kubernetes controller that bridges FluxCD reconciliations to the GitHub Deployments API.",
  icon: "/favicon.svg",
  logoText: "GitHub Deployment Bridge",
  // Project Pages live at https://<user>.github.io/<repo>/
  base: isGitHubActions ? "/github-deployment-bridge/" : "/",
  lang: "en",
  plugins: [
    mermaid({
      mermaidConfig: {
        theme: "base",
        themeVariables: {
          primaryColor: "#d5ebe9",
          primaryTextColor: "#14181c",
          primaryBorderColor: "#1f6f6a",
          lineColor: "#4a535c",
          secondaryColor: "#ebe8e2",
          tertiaryColor: "#f4f2ee",
          fontFamily: "Instrument Sans, ui-sans-serif, system-ui, sans-serif",
        },
        flowchart: { curve: "basis" },
      },
    }),
  ],
  markdown: {
    link: {
      checkDeadLinks: true,
    },
  },
  route: {
    cleanUrls: true,
  },
  globalStyles: path.join(__dirname, "styles/index.css"),
  themeConfig: {
    socialLinks: [
      {
        icon: "github",
        mode: "link",
        content: "https://github.com/roberteggl/github-deployment-bridge",
      },
    ],
    editLink: {
      docRepoBaseUrl:
        "https://github.com/roberteggl/github-deployment-bridge/tree/main/docs",
    },
    lastUpdated: true,
    enableScrollToTop: true,
    footer: {
      message:
        'Released under the <a href="https://github.com/roberteggl/github-deployment-bridge/blob/main/LICENSE">Apache License 2.0</a>.',
    },
  },
  builderConfig: {
    html: {
      tags: [
        {
          tag: "link",
          attrs: {
            rel: "preconnect",
            href: "https://fonts.googleapis.com",
          },
        },
        {
          tag: "link",
          attrs: {
            rel: "preconnect",
            href: "https://fonts.gstatic.com",
            crossorigin: true,
          },
        },
        {
          tag: "link",
          attrs: {
            rel: "stylesheet",
            href: "https://fonts.googleapis.com/css2?family=IBM+Plex+Mono:wght@400;500&family=Instrument+Sans:ital,wght@0,400;0,500;0,600;0,700;1,400&display=swap",
          },
        },
        {
          tag: "meta",
          attrs: {
            name: "theme-color",
            content: "#1f6f6a",
          },
        },
      ],
    },
  },
};

export default defineConfig(config);
