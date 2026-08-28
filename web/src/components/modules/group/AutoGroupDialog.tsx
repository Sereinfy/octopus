import { useState } from 'react';
import { HelpCircle, Save, Sparkles } from 'lucide-react';
import { toast } from 'sonner';
import { useTranslations } from 'use-intl';
import { AutoGroupType } from '@/api/channel';
import {
    useAutoGroupConfig,
    useUpdateAutoGroupConfig,
} from '@/api/group';
import { Button, buttonVariants } from '@/components/ui/button';
import {
    MorphingDialog,
    MorphingDialogClose,
    MorphingDialogContainer,
    MorphingDialogContent,
    MorphingDialogDescription,
    MorphingDialogTitle,
    MorphingDialogTrigger,
    useMorphingDialog,
} from '@/components/ui/morphing-dialog';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { Switch } from '@/components/ui/switch';
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip';

const modes = [AutoGroupType.None, AutoGroupType.Fuzzy, AutoGroupType.Exact];

function SettingLabel({ label, hint }: { label: string; hint: string }) {
    return (
        <span className="flex items-center gap-1.5 text-sm font-medium text-foreground">
            {label}
            <Tooltip>
                <TooltipTrigger asChild>
                    <HelpCircle className="size-3.5 cursor-help text-muted-foreground" />
                </TooltipTrigger>
                <TooltipContent side="top" className="max-w-xs">{hint}</TooltipContent>
            </Tooltip>
        </span>
    );
}

function AutoGroupDialogContent() {
    const t = useTranslations('group.autoGroup');
    const { setIsOpen } = useMorphingDialog();
    const { data: config, isLoading, error } = useAutoGroupConfig();
    const updateConfig = useUpdateAutoGroupConfig();
    const [globalModeOverride, setGlobalModeOverride] = useState<AutoGroupType | null>(null);
    const [createMissingOverride, setCreateMissingOverride] = useState<boolean | null>(null);
    const [normalizeNamesOverride, setNormalizeNamesOverride] = useState<boolean | null>(null);
    const [sourceModes, setSourceModes] = useState<Record<number, AutoGroupType>>({});

    const globalMode = globalModeOverride ?? config?.global_mode ?? AutoGroupType.None;
    const createMissing = createMissingOverride ?? config?.create_missing_groups ?? false;
    const normalizeNames = normalizeNamesOverride ?? config?.normalize_model_names ?? false;

    const modeLabel = (mode: AutoGroupType) => t(`mode.${mode}`);
    const isPending = updateConfig.isPending;

    const save = (runNow: boolean) => {
        if (!config) return;
        updateConfig.mutate({
            global_mode: globalMode,
            create_missing_groups: createMissing,
            normalize_model_names: normalizeNames,
            items: config.sources.map((source) => ({
                channel_id: source.channel_id,
                auto_group: sourceModes[source.channel_id] ?? source.auto_group,
            })),
            run_now: runNow,
        }, {
            onSuccess: () => {
                toast.success(runNow ? t('toast.savedAndRun') : t('toast.saved'));
                setIsOpen(false);
            },
            onError: (saveError) => toast.error(t('toast.failed'), { description: saveError.message }),
        });
    };

    return (
        <div className="flex h-[min(44rem,calc(90vh-2rem))] w-full flex-col">
            <MorphingDialogTitle className="shrink-0">
                <header className="mb-4 flex items-center justify-between">
                    <div>
                        <h2 className="text-2xl font-bold text-card-foreground">{t('title')}</h2>
                        <p className="mt-1 text-sm text-muted-foreground">{t('description')}</p>
                    </div>
                    <MorphingDialogClose className="relative right-0 top-0" />
                </header>
            </MorphingDialogTitle>

            <MorphingDialogDescription className="min-h-0 flex-1 overflow-y-auto pr-1">
                {isLoading && <div className="py-12 text-center text-sm text-muted-foreground">{t('loading')}</div>}
                {error && <div className="py-12 text-center text-sm text-destructive">{error.message}</div>}
                {config && (
                    <TooltipProvider>
                    <div className="space-y-5">
                        <section className="space-y-3 rounded-xl border border-border/50 bg-muted/30 p-3">
                            <div className="flex items-center justify-between gap-4">
                                <SettingLabel label={t('globalMode')} hint={t('globalModeHint')} />
                                <Select
                                    value={String(globalMode)}
                                    onValueChange={(value) => {
                                        const mode = Number(value) as AutoGroupType;
                                        setGlobalModeOverride(mode);
                                        setSourceModes(Object.fromEntries(config.sources.map((source) => [source.channel_id, mode])));
                                    }}
                                >
                                    <SelectTrigger className="h-8 w-36 rounded-xl bg-background text-xs"><SelectValue /></SelectTrigger>
                                    <SelectContent>
                                        {modes.map((mode) => <SelectItem key={mode} value={String(mode)}>{modeLabel(mode)}</SelectItem>)}
                                    </SelectContent>
                                </Select>
                            </div>
                            <div className="flex items-center justify-between gap-4 border-t border-border/40 pt-3">
                                <SettingLabel label={t('normalize')} hint={t('normalizeHint')} />
                                    <Switch checked={normalizeNames} onCheckedChange={setNormalizeNamesOverride} />
                            </div>
                            <div className="flex items-center justify-between gap-4 border-t border-border/40 pt-3">
                                <SettingLabel label={t('createMissing')} hint={t('createMissingHint')} />
                                    <Switch checked={createMissing} onCheckedChange={setCreateMissingOverride} />
                            </div>
                        </section>

                        <section className="space-y-2">
                            <div>
                                <h3 className="text-sm font-medium">{t('channels')}</h3>
                                <p className="text-xs text-muted-foreground">{t('channelsHint')}</p>
                            </div>
                            <div className="divide-y divide-border rounded-lg border border-border">
                                {config.sources.map((source) => (
                                    <div key={source.channel_id} className="grid items-center gap-3 px-3 py-3 sm:grid-cols-[minmax(0,1fr)_10rem]">
                                        <div className="min-w-0">
                                            <div className="flex items-center gap-2">
                                                <span className="truncate text-sm font-medium">{source.channel_name}</span>
                                                {!source.enabled && <span className="text-xs text-muted-foreground">{t('disabled')}</span>}
                                            </div>
                                            <p className="text-xs text-muted-foreground">{t('modelCount', { count: source.model_count })}</p>
                                        </div>
                                        <Select
                                            value={String(sourceModes[source.channel_id] ?? source.auto_group)}
                                            onValueChange={(value) => setSourceModes((current) => ({
                                                ...current,
                                                [source.channel_id]: Number(value) as AutoGroupType,
                                            }))}
                                        >
                                            <SelectTrigger className="w-full rounded-lg"><SelectValue /></SelectTrigger>
                                            <SelectContent>
                                                {modes.map((mode) => <SelectItem key={mode} value={String(mode)}>{modeLabel(mode)}</SelectItem>)}
                                            </SelectContent>
                                        </Select>
                                    </div>
                                ))}
                            </div>
                        </section>
                    </div>
                    </TooltipProvider>
                )}
            </MorphingDialogDescription>

            <div className="mt-4 flex shrink-0 flex-col-reverse gap-2 border-t border-border pt-4 sm:flex-row sm:justify-end">
                <Button type="button" variant="secondary" className="rounded-xl" onClick={() => setIsOpen(false)}>{t('cancel')}</Button>
                <Button type="button" variant="outline" className="rounded-xl" disabled={!config || isPending} onClick={() => save(false)}>
                    <Save className="size-4" />{t('save')}
                </Button>
                <Button type="button" className="rounded-xl" disabled={!config || isPending} onClick={() => save(true)}>
                    <Sparkles className="size-4" />{isPending ? t('running') : t('saveAndRun')}
                </Button>
            </div>
        </div>
    );
}

export function AutoGroupDialog() {
    const t = useTranslations('group.autoGroup');
    return (
        <MorphingDialog>
            <MorphingDialogTrigger
                className={buttonVariants({
                    variant: 'ghost',
                    size: 'icon',
                    className: 'rounded-xl transition-none hover:bg-transparent text-muted-foreground hover:text-foreground',
                })}
            >
                <Sparkles className="size-4" />
                <span className="sr-only">{t('title')}</span>
            </MorphingDialogTrigger>
            <MorphingDialogContainer>
                <MorphingDialogContent className="relative flex w-full md:max-w-3xl flex-col overflow-hidden rounded-3xl bg-card px-4 py-2 text-card-foreground max-h-[90vh]">
                    <AutoGroupDialogContent />
                </MorphingDialogContent>
            </MorphingDialogContainer>
        </MorphingDialog>
    );
}
