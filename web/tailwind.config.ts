import type { Config } from 'tailwindcss';

const config: Config = {
  content: ['./src/components/**/*.{js,ts,jsx,tsx,mdx}', './src/app/**/*.{js,ts,jsx,tsx,mdx}'],
  theme: {
    extend: {
      container: {
        center: true,
        padding: '30px'
      },
      colors: {
        success: 'hsl(var(--success))',
        warning: 'hsl(var(--warning))',
        danger: 'hsl(var(--danger))',
        pending: 'hsl(var(--pending))',
        active: 'hsl(var(--active))',
        succeeded: 'hsl(var(--succeeded))',
        executed: 'hsl(var(--executed))',
        defeated: 'hsl(var(--defeated))',
        canceled: 'hsl(var(--canceled))'
      }
    }
  }
};
export default config;
