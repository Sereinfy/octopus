import { useStatsSummary } from '@/api/stats';
import { ChartContainer, ChartTooltip, ChartTooltipContent } from '@/components/ui/chart';
import { AnimatedNumber } from '@/components/common/AnimatedNumber';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { useHomeViewStore, type ChartPeriod } from '@/components/modules/home/store';
import { formatCount, formatMoney, formatTime } from '@/lib/utils';
import { Area, AreaChart, CartesianGrid, XAxis, YAxis } from 'recharts';
import { useTranslations } from 'use-intl';

const PERIODS: readonly ChartPeriod[] = ['1', '7', '30', 'all'];
const PERIOD_KEY: Record<ChartPeriod, 'today' | 'last7Days' | 'last30Days' | 'allTime'> = {
    '1': 'today',
    '7': 'last7Days',
    '30': 'last30Days',
    all: 'allTime',
};

function formatDate(date: string): string {
    if (date.length !== 8) return date;
    return `${date.slice(4, 6)}/${date.slice(6, 8)}`;
}

export function StatsChart() {
    const period = useHomeViewStore((state) => state.chartPeriod);
    const setChartPeriod = useHomeViewStore((state) => state.setChartPeriod);
    const { data: summary } = useStatsSummary(period);
    const t = useTranslations('home.summary');

    const hero = summary?.total_cost.formatted;
    const metrics = summary
        ? {
            requests: summary.request_count.formatted,
            inputTokens: summary.input_token.formatted,
            outputTokens: summary.output_token.formatted,
            cacheHits: summary.cache_read_token.formatted,
            cacheHitRate: summary.cache_hit_rate.formatted,
            totalTokens: summary.total_token.formatted,
            waitTime: summary.wait_time.formatted,
        }
        : {
            requests: formatCount(0).formatted,
            inputTokens: formatCount(0).formatted,
            outputTokens: formatCount(0).formatted,
            cacheHits: formatCount(0).formatted,
            cacheHitRate: { value: '0.0%', unit: '' },
            totalTokens: formatCount(0).formatted,
            waitTime: formatTime(0).formatted,
        };
    const chartData = summary?.points.map((point) => ({
        date: formatDate(point.date),
        total_cost: point.total_cost,
    })) ?? [];
    const heroUnitSuffix = hero?.unit.endsWith('$') ? hero.unit.slice(0, -1) : hero?.unit;

    return (
        <section className="rounded-3xl border border-border bg-card text-card-foreground">
            <header className="flex flex-col gap-4 px-5 pb-4 pt-5 md:flex-row md:items-start md:justify-between">
                <div>
                    <p className="text-xs text-muted-foreground">{t(`headline.${PERIOD_KEY[period]}`)}</p>
                    <p className="mt-1 text-4xl font-semibold tabular-nums md:text-5xl">
                        {!hero ? (
                            <span className="text-muted-foreground">—</span>
                        ) : (
                            <>
                                <span className="mr-1 text-2xl text-muted-foreground">$</span>
                                <AnimatedNumber value={hero.value} />
                                {heroUnitSuffix && <span className="ml-1 text-xl text-muted-foreground">{heroUnitSuffix}</span>}
                            </>
                        )}
                    </p>
                </div>
                <Tabs value={period} onValueChange={(value) => setChartPeriod(value as ChartPeriod)}>
                    <TabsList className="w-full sm:w-auto">
                        {PERIODS.map((value) => (
                            <TabsTrigger key={value} value={value}>
                                {t(`periods.${PERIOD_KEY[value]}`)}
                            </TabsTrigger>
                        ))}
                    </TabsList>
                </Tabs>
            </header>

            <div className="mx-5 flex flex-wrap items-baseline gap-x-6 gap-y-2 border-t border-border/60 py-3 text-sm tabular-nums">
                <StatItem label={t('metrics.requests')} value={metrics.requests} />
                <span className="hidden h-4 w-px bg-border/60 sm:inline-block" />
                <StatItem label={t('metrics.inputTokens')} value={metrics.inputTokens} />
                <span className="hidden h-4 w-px bg-border/60 sm:inline-block" />
                <StatItem label={t('metrics.outputTokens')} value={metrics.outputTokens} />
                <span className="hidden h-4 w-px bg-border/60 sm:inline-block" />
                <StatItem label={t('metrics.cacheHits')} value={metrics.cacheHits} />
                <StatItem label={t('metrics.cacheHitRate')} value={metrics.cacheHitRate} />
                <span className="hidden h-4 w-px bg-border/60 sm:inline-block" />
                <StatItem label={t('metrics.totalTokens')} value={metrics.totalTokens} />
                <span className="hidden h-4 w-px bg-border/60 sm:inline-block" />
                <StatItem label={t('metrics.waitTime')} value={metrics.waitTime} />
            </div>

            <ChartContainer config={{ total_cost: { label: t('headline.allTime') } }} className="h-40 w-full">
                <AreaChart accessibilityLayer data={chartData}>
                    <defs>
                        <linearGradient id="fillCost" x1="0" y1="0" x2="0" y2="1">
                            <stop offset="5%" stopColor="var(--chart-1)" stopOpacity={0.35} />
                            <stop offset="95%" stopColor="var(--chart-1)" stopOpacity={0.05} />
                        </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" vertical={false} />
                    <XAxis dataKey="date" tickLine={false} axisLine={false} />
                    <YAxis
                        tickLine={false}
                        axisLine={false}
                        tickFormatter={(value) => {
                            const formatted = formatMoney(value);
                            return `${formatted.formatted.value}${formatted.formatted.unit}`;
                        }}
                    />
                    <ChartTooltip cursor={false} content={<ChartTooltipContent indicator="line" />} />
                    <Area type="monotone" dataKey="total_cost" stroke="var(--chart-1)" fill="url(#fillCost)" />
                </AreaChart>
            </ChartContainer>
        </section>
    );
}

function StatItem({ label, value }: { label: string; value: ReturnType<typeof formatCount>['formatted'] }) {
    return (
        <div className="flex items-baseline gap-1.5">
            <span className="text-xs text-muted-foreground">{label}</span>
            <span className="font-medium">
                <AnimatedNumber value={value.value} />
                {value.unit && <span className="ml-0.5 text-xs text-muted-foreground">{value.unit}</span>}
            </span>
        </div>
    );
}
