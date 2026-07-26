// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

import * as path from "node:path";
import { defineConfig, type UserConfig } from "@rspress/core";
import mermaid from "rspress-plugin-mermaid";

const config: UserConfig = {
  root: path.join(__dirname, "docs"),
  title: "GitHub Deployment Bridge",
  description:
    "Lightweight Kubernetes controller that bridges FluxCD reconciliations to the GitHub Deployments API.",
  icon: "/favicon.svg",
  logoText: "GitHub Deployment Bridge",
  // Custom domain: https://deployment-bridge.eggl.dev (root path, not /<repo>/)
  base: "/",
  lang: "en",
  head: [
    ["meta", { name: "author", content: "Robert Eggl" }],
    ["link", { rel: "author", href: "https://eggl.dev" }],
    [
      "link",
      {
        rel: "alternate",
        title: "Artifact Hub",
        href: "https://artifacthub.io/packages/helm/github-deployment-bridge/github-deployment-bridge",
      },
    ],
  ],
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
        'Copyright © 2026 Robert Eggl. Released under the <a href="https://github.com/roberteggl/github-deployment-bridge/blob/main/LICENSE">Apache License 2.0</a>.',
    },
  },
  builderConfig: {
    html: {
      tags: [
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
