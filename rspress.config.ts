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
        theme: "neutral",
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
          tag: "meta",
          attrs: {
            name: "theme-color",
            content: "#0b1020",
          },
        },
      ],
    },
  },
};

export default defineConfig(config);
