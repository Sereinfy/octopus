import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export type Locale = 'zh_hans' | 'zh_hant' | 'en';

/** Classic shadcn-style color palettes (independent of light/dark mode). */
export type ColorTheme =
    | 'default'
    | 'zinc'
    | 'slate'
    | 'stone'
    | 'blue'
    | 'green'
    | 'orange'
    | 'rose'
    | 'violet';

export const COLOR_THEMES: { id: ColorTheme; swatch: string }[] = [
    { id: 'default', swatch: 'oklch(0.62 0.12 145)' },
    { id: 'zinc', swatch: 'oklch(0.45 0.02 260)' },
    { id: 'slate', swatch: 'oklch(0.48 0.04 250)' },
    { id: 'stone', swatch: 'oklch(0.50 0.02 60)' },
    { id: 'blue', swatch: 'oklch(0.55 0.18 255)' },
    { id: 'green', swatch: 'oklch(0.58 0.15 150)' },
    { id: 'orange', swatch: 'oklch(0.65 0.18 45)' },
    { id: 'rose', swatch: 'oklch(0.58 0.18 15)' },
    { id: 'violet', swatch: 'oklch(0.55 0.20 295)' },
];

interface SettingState {
    locale: Locale;
    colorTheme: ColorTheme;
    setLocale: (locale: Locale) => void;
    setColorTheme: (theme: ColorTheme) => void;
}

export const useSettingStore = create<SettingState>()(
    persist(
        (set) => ({
            locale: 'zh_hans',
            colorTheme: 'default',
            setLocale: (locale) => set({ locale }),
            setColorTheme: (colorTheme) => set({ colorTheme }),
        }),
        {
            name: 'octopus-settings',
        }
    )
);
