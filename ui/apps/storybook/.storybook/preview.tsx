import type { Preview } from "@storybook/react";
import { withThemeByClassName } from "@storybook/addon-themes";
import React from "react";
import { MockAuthProvider } from "../src/mocks/auth-provider";
import "../src/styles.css";

const preview: Preview = {
  parameters: {
    controls: {
      matchers: {
        color: /(background|color)$/i,
        date: /Date$/i,
      },
    },
    layout: "centered",
    backgrounds: {
      default: "light",
      values: [
        { name: "light", value: "#ffffff" },
        { name: "dark", value: "#09090b" },
        { name: "muted", value: "#f4f4f5" },
      ],
    },
    viewport: {
      viewports: {
        mobile: { name: "Mobile", styles: { width: "375px", height: "812px" } },
        tablet: { name: "Tablet", styles: { width: "768px", height: "1024px" } },
        desktop: { name: "Desktop", styles: { width: "1280px", height: "800px" } },
      },
    },
  },
  decorators: [
    withThemeByClassName({
      themes: {
        light: "",
        dark: "dark",
      },
      defaultTheme: "light",
    }),
    (Story, context) => {
      // Skip the container wrapper for fullscreen stories
      if (context.parameters.layout === "fullscreen") {
        return (
          <MockAuthProvider>
            <Story />
          </MockAuthProvider>
        );
      }

      return (
        <MockAuthProvider>
          <div className="flex min-h-[600px] min-w-[480px] items-center justify-center p-8">
            <Story />
          </div>
        </MockAuthProvider>
      );
    },
  ],
};

export default preview;
