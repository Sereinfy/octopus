import { useState } from 'react';
import { Save, WandSparkles } from 'lucide-react';
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

const modes = [AutoGroupType.None, AutoGroupType.Fuzzy, AutoGroupType.Exact, AutoGroupType.Regex];

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
        <div className="flex h-[min(42rem,calc(100vh-2rem))] w-screen max-w-3xl flex-col">
            <MorphingDialogTitle className="shrink-0">
                <header className="mb-4 flex items-center justify-between">
                    <div>
                        <h2 className="text-xl font-semibold text-card-foreground">{t('title')}</h2>
                        <p className="mt-1 text-sm text-muted-foreground">{t('description')}</p>
                    </div>
                    <MorphingDialogClose className="relative right-0 top-0" />
                </header>
            </MorphingDialogTitle>

            <MorphingDialogDescription className="min-h-0 flex-1 overflow-y-auto pr-1">
                {isLoading && <div className="py-12 text-center text-sm text-muted-foreground">{t('loading')}</div>}
                {error && <div className="py-12 text-center text-sm text-destructive">{error.message}</div>}
                {config && (
                    <div className="space-y-5">
                        <section className="grid gap-4 border-b border-border pb-5 md:grid-cols-2">
                            <div className="space-y-2">
                                <label className="text-sm font-medium">{t('globalMode')}</label>
                                <Select
                                    value={String(globalMode)}
                                    onValueChange={(value) => {
                                        const mode = Number(value) as AutoGroupType;
                                        setGlobalModeOverride(mode);
                                        setSourceModes(Object.fromEntries(config.sources.map((source) => [source.channel_id, mode])));
                                    }}
                                >
                                    <SelectTrigger className="w-full rounded-lg"><SelectValue /></SelectTrigger>
                                    <SelectContent>
                                        {modes.map((mode) => <SelectItem key={mode} value={String(mode)}>{modeLabel(mode)}</SelectItem>)}
                                    </SelectContent>
                                </Select>
                                <p className="text-xs text-muted-foreground">{t('globalModeHint')}</p>
                            </div>
                            <div className="space-y-3 pt-1">
                                <label className="flex items-center justify-between gap-4">
                                    <span>
                                        <span className="block text-sm font-medium">{t('normalize')}</span>
                                        <span className="block text-xs text-muted-foreground">{t('normalizeHint')}</span>
                                    </span>
                                    <Switch checked={normalizeNames} onCheckedChange={setNormalizeNamesOverride} />
                                </label>
                                <label className="flex items-center justify-between gap-4">
                                    <span>
                                        <span className="block text-sm font-medium">{t('createMissing')}</span>
                                        <span className="block text-xs text-muted-foreground">{t('createMissingHint')}</span>
                                    </span>
                                    <Switch checked={createMissing} onCheckedChange={setCreateMissingOverride} />
                                </label>
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
                )}
            </MorphingDialogDescription>

            <div className="mt-4 flex shrink-0 justify-end gap-2 border-t border-border pt-4">
                <Button type="button" variant="secondary" onClick={() => setIsOpen(false)}>{t('cancel')}</Button>
                <Button type="button" variant="outline" disabled={!config || isPending} onClick={() => save(false)}>
                    <Save />{t('save')}
                </Button>
                <Button type="button" disabled={!config || isPending} onClick={() => save(true)}>
                    <WandSparkles />{isPending ? t('running') : t('saveAndRun')}
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
                <WandSparkles className="size-4" />
                <span className="sr-only">{t('title')}</span>
            </MorphingDialogTrigger>
            <MorphingDialogContainer>
                <MorphingDialogContent className="flex max-h-[calc(100vh-2rem)] w-fit max-w-full flex-col overflow-hidden rounded-2xl bg-card px-6 py-5 text-card-foreground">
                    <AutoGroupDialogContent />
                </MorphingDialogContent>
            </MorphingDialogContainer>
        </MorphingDialog>
    );
}
