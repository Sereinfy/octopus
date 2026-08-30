import { useChannelList } from '@/api/channel';
import { useMemo } from 'react';
import { useTranslations } from 'use-intl';
import { TrendingUp } from 'lucide-react';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { formatCount, formatMoney, formatTime } from '@/lib/utils';
import { useHomeViewStore, type RankSortMode } from '@/components/modules/home/store';
import { formatCacheHitRate, type StatsMetricsFormatted } from '@/api/stats';

interface RankData {
    id: string;
    name: string;
    channelName?: string;
    formatted: StatsMetricsFormatted;
}

interface RankCardProps {
    title: string;
    items: RankData[];
    sortMode: RankSortMode;
    onSortModeChange: (value: RankSortMode) => void;
    hideChannelName: boolean;
}

function RankCard({ title, items, sortMode, onSortModeChange, hideChannelName }: RankCardProps) {
    const t = useTranslations('home.rank');
    const sortField = sortMode === 'cost' ? 'total_cost' : sortMode === 'count' ? 'request_count' : 'total_token';
    const ranked = useMemo(
        () => [...items].sort((a, b) => b.formatted[sortField].raw - a.formatted[sortField].raw),
        [items, sortField]
    );

    const renderList = () => {
        if (ranked.length === 0) {
            return (
                <div className="flex flex-col items-center justify-center py-8 text-muted-foreground">
                    <TrendingUp className="mb-3 h-12 w-12 opacity-30" />
                    <p className="text-sm">{t('noData')}</p>
                </div>
            );
        }

        return (
            <div className="max-h-[300px] space-y-3 overflow-y-auto">
                {ranked.map((item, index) => {
                    const successCount = item.formatted.request_success.raw;
                    const failedCount = item.formatted.request_failed.raw;
                    const totalCount = successCount + failedCount;
                    const successRate = totalCount > 0 ? (successCount / totalCount) * 100 : 0;

                    return (
                        <div key={item.id} className="grid grid-cols-[auto_minmax(0,1fr)_auto] items-center gap-3 py-3">
                            <div className="flex items-center justify-center text-lg font-bold">{index + 1}</div>

                            <div className="min-w-0">
                                <p className={`truncate text-sm font-medium ${hideChannelName && !item.channelName ? 'select-none blur-[3px]' : ''}`}>
                                    {item.name}
                                </p>
                                {item.channelName && (
                                    <p className={`mt-1 truncate text-xs text-muted-foreground ${hideChannelName ? 'select-none blur-[3px]' : ''}`}>
                                        {item.channelName}
                                    </p>
                                )}
                                {sortMode === 'count' && (
                                    <div className="mt-1 flex items-center gap-1 text-xs text-muted-foreground">
                                        <span>{t('successRate')}:</span>
                                        <span>{successRate.toFixed(1)}%</span>
                                    </div>
                                )}
                                {sortMode === 'tokens' && (
                                    <div className="mt-1 flex items-center gap-1 text-xs text-muted-foreground">
                                        <span>{t('cacheHitRate')}:</span>
                                        <span>{item.formatted.cache_hit_rate.formatted.value}</span>
                                    </div>
                                )}
                            </div>

                            <div className="flex items-center gap-1 text-right">
                                {sortMode === 'count' ? (
                                    <div className="flex items-center gap-1 text-sm font-medium tabular-nums">
                                        <span className="text-accent">
                                            {item.formatted.request_success.formatted.value}
                                            <span className="text-xs text-muted-foreground">{item.formatted.request_success.formatted.unit}</span>
                                        </span>
                                        <span className="font-light text-muted-foreground/40">/</span>
                                        <span className="text-destructive">
                                            {item.formatted.request_failed.formatted.value}
                                            <span className="text-xs text-muted-foreground">{item.formatted.request_failed.formatted.unit}</span>
                                        </span>
                                    </div>
                                ) : (
                                    <span className="text-base font-semibold">
                                        {item.formatted[sortField].formatted.value}
                                        <span className="text-xs text-muted-foreground">{item.formatted[sortField].formatted.unit}</span>
                                    </span>
                                )}
                            </div>
                        </div>
                    );
                })}
            </div>
        );
    };

    return (
        <section className="rounded-3xl border border-border bg-card px-4 pt-2 text-card-foreground">
            <Tabs value={sortMode} onValueChange={(value) => onSortModeChange(value as RankSortMode)}>
                <div className="flex flex-wrap items-center justify-between gap-2">
                    <h3 className="text-base font-semibold">{title}</h3>
                    <TabsList variant="text" className="p-0">
                        <TabsTrigger value="cost" className="pr-0">{t('sortByCost')}</TabsTrigger>
                        <span aria-hidden="true" className="mx-1 inline-flex h-full -translate-y-px items-center text-sm font-medium leading-none text-muted-foreground/50">/</span>
                        <TabsTrigger value="count" className="px-0">{t('sortByCount')}</TabsTrigger>
                        <span aria-hidden="true" className="mx-1 inline-flex h-full -translate-y-px items-center text-sm font-medium leading-none text-muted-foreground/50">/</span>
                        <TabsTrigger value="tokens" className="pl-0">{t('sortByTokens')}</TabsTrigger>
                    </TabsList>
                </div>
                <TabsContent value="cost">{sortMode === 'cost' && renderList()}</TabsContent>
                <TabsContent value="count">{sortMode === 'count' && renderList()}</TabsContent>
                <TabsContent value="tokens">{sortMode === 'tokens' && renderList()}</TabsContent>
            </Tabs>
        </section>
    );
}

export function Rank() {
    const { data: channelData } = useChannelList();
    const t = useTranslations('home.rank');
    const channelSortMode = useHomeViewStore((state) => state.channelRankSortMode);
    const setChannelSortMode = useHomeViewStore((state) => state.setChannelRankSortMode);
    const modelSortMode = useHomeViewStore((state) => state.modelRankSortMode);
    const setModelSortMode = useHomeViewStore((state) => state.setModelRankSortMode);
    const isChannelNameHidden = useHomeViewStore((state) => state.isChannelNameHidden);

    const channelItems = useMemo<RankData[]>(() => (channelData ?? []).map((channel) => ({
        id: `channel-${channel.raw.id}`,
        name: channel.raw.name,
        formatted: channel.formatted,
    })), [channelData]);

    const modelItems = useMemo<RankData[]>(() => (channelData ?? []).flatMap((channel) => channel.raw.models.map((model) => ({
        id: `model-${model.id}`,
        name: model.name,
        channelName: channel.raw.name,
        formatted: {
            input_token: formatCount(model.input_token),
            output_token: formatCount(model.output_token),
            cache_read_token: formatCount(model.cache_read_token),
            cache_write_token: formatCount(model.cache_write_token),
            cache_hit_rate: formatCacheHitRate(model.input_token, model.cache_read_token),
            total_token: formatCount(model.input_token + model.output_token),
            input_cost: formatMoney(model.input_cost),
            output_cost: formatMoney(model.output_cost),
            total_cost: formatMoney(model.input_cost + model.output_cost),
            wait_time: formatTime(model.wait_time),
            request_success: formatCount(model.request_success),
            request_failed: formatCount(model.request_failed),
            request_count: formatCount(model.request_success + model.request_failed),
        },
    }))), [channelData]);

    return (
        <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
            <RankCard
                title={t('channel')}
                items={channelItems}
                sortMode={channelSortMode}
                onSortModeChange={setChannelSortMode}
                hideChannelName={isChannelNameHidden}
            />
            <RankCard
                title={t('model')}
                items={modelItems}
                sortMode={modelSortMode}
                onSortModeChange={setModelSortMode}
                hideChannelName={isChannelNameHidden}
            />
        </div>
    );
}
