// SPDX-FileCopyrightText: 2026 Robert Eggl <robert@eggl.dev>
//
// SPDX-License-Identifier: Apache-2.0

import {
  HomeBackground,
  HomeFeature,
  HomeFooter,
  HomeHero,
  Layout as BasicLayout,
  renderHtmlOrText,
  type HomeLayoutProps,
} from "@rspress/core/theme-original";
import { useSite } from "@rspress/core/runtime";
import { HeroBridgeGraphic } from "./HeroBridgeGraphic";

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

function HomeLayout(props: HomeLayoutProps) {
  const {
    beforeHero,
    afterHero,
    beforeFeatures,
    afterFeatures,
    beforeHeroActions,
    afterHeroActions,
  } = props;

  return (
    <>
      <HomeBackground />
      {beforeHero}
      <HomeHero
        beforeHeroActions={beforeHeroActions}
        afterHeroActions={afterHeroActions}
        image={<HeroBridgeGraphic />}
      />
      {afterHero}
      {beforeFeatures}
      <HomeFeature />
      {afterFeatures}
      <HomeFooter />
    </>
  );
}

const Layout = () => (
  <BasicLayout afterDocFooter={<SiteFooterMessage />} />
);

export * from "@rspress/core/theme-original";
export { Layout, HomeLayout };
