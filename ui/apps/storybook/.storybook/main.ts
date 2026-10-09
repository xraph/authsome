import type { StorybookConfig } from "@storybook/react-vite";
import tailwindcss from "@tailwindcss/vite";

const config: StorybookConfig = {
  stories: ["../src/stories/**/*.stories.@(ts|tsx)"],
  addons: [
    "@storybook/addon-essentials",
    "@storybook/addon-themes",
    "@storybook/addon-a11y",
  ],
  framework: {
    name: "@storybook/react-vite",
    options: {},
  },
  viteFinal: async (config) => {
    config.plugins = config.plugins || [];
    config.plugins.push(tailwindcss());
    const target = ["chrome87", "edge88", "es2020", "firefox78", "safari14.1"];
    config.build = { ...config.build, target };
    config.optimizeDeps = {
      ...config.optimizeDeps,
      esbuildOptions: { ...config.optimizeDeps?.esbuildOptions, target },
    };
    return config;
  },
};

export default config;
