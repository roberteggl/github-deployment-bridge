// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

import {
  Layout as BasicLayout,
  renderHtmlOrText,
} from "@rspress/core/theme-original";
import { useSite } from "@rspress/core/runtime";

function SiteFooterMessage() {
  const { site } = useSite();
  const message = site.themeConfig.footer?.message;
  if (!message) {
    return null;
  }

  return (
    <div className="gdb-site-footer">
      <div
        className="gdb-site-footer__message"
        {...renderHtmlOrText(message)}
      />
    </div>
  );
}

const Layout = () => (
  <BasicLayout afterDocFooter={<SiteFooterMessage />} />
);

export { Layout };
export * from "@rspress/core/theme-original";
