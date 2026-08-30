import { createContext, useContext, useEffect, useState, type ReactNode } from 'react';
import { useSettingStore, type ColorTheme } from '@/stores/setting';

type Theme = 'light' | 'dark' | 'system';
type ResolvedTheme = Exclude<Theme, 'system'>;

interface ThemeContextValue {
    theme: Theme;
    resolvedTheme: ResolvedTheme;
    setTheme: (theme: string) => void;
}

const META_BY_THEME: Record<ColorTheme, { light: string; dark: string }> = {
    default: { light: '#eae9e3', dark: '#413a2c' },
    zinc: { light: '#fafafa', dark: '#18181b' },
    slate: { light: '#f8fafc', dark: '#0f172a' },
    stone: { light: '#fafaf9', dark: '#1c1917' },
    blue: { light: '#f8fafc', dark: '#0b1220' },
    green: { light: '#f7fdf9', dark: '#0c1a12' },
    orange: { light: '#fffaf5', dark: '#1c120a' },
    rose: { light: '#fff7f8', dark: '#1a0c10' },
    violet: { light: '#faf8ff', dark: '#120f1c' },
};

const ThemeContext = createContext<ThemeContextValue | null>(null);

export function ThemeProvider({ children }: { children: ReactNode }) {
    const [theme, setThemeState] = useState<Theme>(() => {
        const storedTheme = localStorage.getItem('theme');
        return storedTheme === 'light' || storedTheme === 'dark' || storedTheme === 'system'
            ? storedTheme
            : 'system';
    });
    const [systemTheme, setSystemTheme] = useState<ResolvedTheme>(() =>
        window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
    );
    const colorTheme = useSettingStore((state) => state.colorTheme);
    const resolvedTheme = theme === 'system' ? systemTheme : theme;

    useEffect(() => {
        const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');

        const handleChange = (event: MediaQueryListEvent) => {
            setSystemTheme(event.matches ? 'dark' : 'light');
        };

        mediaQuery.addEventListener('change', handleChange);
        return () => mediaQuery.removeEventListener('change', handleChange);
    }, []);

    useEffect(() => {
        const root = document.documentElement;
        if (colorTheme !== 'default') {
            root.setAttribute('data-color-theme', colorTheme);
        } else {
            root.removeAttribute('data-color-theme');
        }

        root.classList.toggle('dark', resolvedTheme === 'dark');
        root.style.colorScheme = resolvedTheme;
        document.querySelector('meta[name="theme-color"]')?.setAttribute(
            'content',
            (META_BY_THEME[colorTheme] ?? META_BY_THEME.default)[resolvedTheme]
        );
        localStorage.setItem('theme', theme);
    }, [colorTheme, resolvedTheme, theme]);

    const setTheme = (value: string) => {
        if (value === 'light' || value === 'dark' || value === 'system') {
            setThemeState(value);
        }
    };

    return (
        <ThemeContext value={{ theme, resolvedTheme, setTheme }}>
            {children}
        </ThemeContext>
    );
}

export function useTheme() {
    const context = useContext(ThemeContext);
    if (!context) {
        throw new Error('useTheme must be used within ThemeProvider');
    }
    return context;
}
