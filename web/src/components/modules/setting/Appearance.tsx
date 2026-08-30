import { useTheme } from '@/provider/theme';
import { useTranslations } from 'use-intl';
import { Sun, Moon, Monitor, Languages, Palette, Check } from 'lucide-react';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import {
    COLOR_THEMES,
    useSettingStore,
    type ColorTheme,
    type Locale,
} from '@/stores/setting';
import { cn } from '@/lib/utils';

export function SettingAppearance() {
    const t = useTranslations('setting');
    const { theme, setTheme } = useTheme();
    const { locale, setLocale, colorTheme, setColorTheme } = useSettingStore();

    return (
        <div className="rounded-3xl border border-border bg-card p-6 space-y-5">
            <h2 className="text-lg font-bold text-card-foreground flex items-center gap-2">
                <Sun className="h-5 w-5" />
                {t('appearance')}
            </h2>

            {/* Light/dark mode */}
            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                    {theme === 'dark' ? <Moon className="h-5 w-5 text-muted-foreground" /> : <Sun className="h-5 w-5 text-muted-foreground" />}
                    <span className="text-sm font-medium">{t('theme.label')}</span>
                </div>
                <Select value={theme} onValueChange={setTheme}>
                    <SelectTrigger className="w-36 rounded-xl">
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent className="rounded-xl">
                        <SelectItem value="light" className="rounded-xl">
                            <Sun className="size-4" />
                            {t('theme.light')}
                        </SelectItem>
                        <SelectItem value="dark" className="rounded-xl">
                            <Moon className="size-4" />
                            {t('theme.dark')}
                        </SelectItem>
                        <SelectItem value="system" className="rounded-xl">
                            <Monitor className="size-4" />
                            {t('theme.system')}
                        </SelectItem>
                    </SelectContent>
                </Select>
            </div>

            {/* Classic shadcn color palettes */}
            <div className="space-y-3">
                <div className="flex items-center gap-3">
                    <Palette className="h-5 w-5 text-muted-foreground" />
                    <div>
                        <div className="text-sm font-medium">{t('colorTheme.label')}</div>
                        <div className="text-xs text-muted-foreground">{t('colorTheme.hint')}</div>
                    </div>
                </div>
                <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
                    {COLOR_THEMES.map((item) => {
                        const active = colorTheme === item.id;
                        return (
                            <button
                                key={item.id}
                                type="button"
                                onClick={() => setColorTheme(item.id as ColorTheme)}
                                className={cn(
                                    'flex items-center gap-2.5 rounded-2xl border px-3 py-2.5 text-left transition',
                                    active
                                        ? 'border-primary/40 bg-primary/10 ring-2 ring-primary/20'
                                        : 'border-border/70 bg-background/60 hover:bg-muted/50',
                                )}
                            >
                                <span
                                    className="size-5 shrink-0 rounded-full border border-black/10 shadow-sm dark:border-white/10"
                                    style={{ background: item.swatch }}
                                    aria-hidden
                                />
                                <span className="min-w-0 flex-1 truncate text-sm font-medium">
                                    {t(`colorTheme.names.${item.id}`)}
                                </span>
                                {active ? <Check className="size-4 shrink-0 text-primary" /> : null}
                            </button>
                        );
                    })}
                </div>
            </div>

            {/* Language */}
            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                    <Languages className="h-5 w-5 text-muted-foreground" />
                    <span className="text-sm font-medium">{t('language.label')}</span>
                </div>
                <Select value={locale} onValueChange={(v) => setLocale(v as Locale)}>
                    <SelectTrigger className="w-36 rounded-xl">
                        <SelectValue />
                    </SelectTrigger>
                    <SelectContent className="rounded-xl">
                        <SelectItem value="zh_hans" className="rounded-xl">{t('language.zh_hans')}</SelectItem>
                        <SelectItem value="zh_hant" className="rounded-xl">{t('language.zh_hant')}</SelectItem>
                        <SelectItem value="en" className="rounded-xl">{t('language.en')}</SelectItem>
                    </SelectContent>
                </Select>
            </div>
        </div>
    );
}
