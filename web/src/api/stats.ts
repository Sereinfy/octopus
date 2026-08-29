import { queryOptions, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { apiRequest } from './client';
import { statsDailyQueryOptions, statsHourlyQueryOptions, statsSummaryQueryOptions, statsTodayQueryOptions, statsTotalQueryOptions } from './queries';
import { formatCount, formatMoney, formatTime } from '@/lib/utils';

/**
 * 统计数据
 */
export interface StatsMetrics {
    input_token: number;
    output_token: number;
    cache_read_token: number;
    cache_write_token: number;
    input_cost: number;
    output_cost: number;
    wait_time: number;
    request_success: number;
    request_failed: number;
}

export interface StatsMetricsFormatted {
    input_token: ReturnType<typeof formatCount>;
    output_token: ReturnType<typeof formatCount>;
    cache_read_token: ReturnType<typeof formatCount>;
    cache_write_token: ReturnType<typeof formatCount>;
    cache_hit_rate: ReturnType<typeof formatPercentage>;
    input_cost: ReturnType<typeof formatMoney>;
    output_cost: ReturnType<typeof formatMoney>;
    wait_time: ReturnType<typeof formatTime>;
    request_success: ReturnType<typeof formatCount>;
    request_failed: ReturnType<typeof formatCount>;

    request_count: ReturnType<typeof formatCount>;
    total_token: ReturnType<typeof formatCount>;
    total_cost: ReturnType<typeof formatMoney>;
}

export interface StatsDaily extends StatsMetrics {
    date: string;
}
export interface StatsDailyResponse {
    max_request_count: number;
    items: StatsDaily[];
}
export interface StatsDailyFormatted extends StatsMetricsFormatted {
    date: string;
}

export function formatPercentage(value: number | undefined) {
    const raw = value ?? 0;
    return { raw, formatted: { value: `${raw.toFixed(1)}%`, unit: '' } };
}

export function formatCacheHitRate(inputToken: number | undefined, cacheReadToken: number | undefined) {
    const input = Math.max(0, inputToken ?? 0);
    const cached = Math.max(0, cacheReadToken ?? 0);
    const rate = input + cached > 0 ? (cached / (input + cached)) * 100 : 0;
    return formatPercentage(rate);
}
interface StatsDailyFormattedResponse {
    max_request_count: number;
    items: StatsDailyFormatted[];
}

export interface StatsTotal extends StatsMetrics {
    id: number;
}
type StatsTotalFormatted = StatsMetricsFormatted;

export interface StatsSummaryPoint {
    date: string;
    total_cost: number;
}

export interface StatsSummary extends StatsMetrics {
    period: '1' | '7' | '30' | 'all';
    points: StatsSummaryPoint[];
}

export interface StatsSummaryFormatted extends StatsMetricsFormatted {
    period: StatsSummary['period'];
    points: StatsSummaryPoint[];
}

export interface StatsHourly extends StatsMetrics {
    hour: number;
    date: string;
}
interface StatsHourlyFormatted extends StatsMetricsFormatted {
    hour: number;
    date: string;
}

const formatStatsDaily = (item: StatsDaily): StatsDailyFormatted => ({
    input_token: formatCount(item.input_token),
    output_token: formatCount(item.output_token),
    cache_read_token: formatCount(item.cache_read_token),
    cache_write_token: formatCount(item.cache_write_token),
    cache_hit_rate: formatCacheHitRate(item.input_token, item.cache_read_token),
    total_token: formatCount(item.input_token + item.output_token),
    input_cost: formatMoney(item.input_cost),
    output_cost: formatMoney(item.output_cost),
    total_cost: formatMoney(item.input_cost + item.output_cost),
    wait_time: formatTime(item.wait_time),
    request_success: formatCount(item.request_success),
    request_failed: formatCount(item.request_failed),
    request_count: formatCount(item.request_success + item.request_failed),
    date: item.date,
});
/**
 * API Key 统计数据
 */
export interface StatsAPIKey extends StatsMetrics {
    api_key_id: number;
}

export interface StatsAPIKeyFormatted extends StatsMetricsFormatted {
    api_key_id: number;
}

// statsDailyFormattedQueryOptions 统一首页每日统计查询、格式化和刷新策略。
const statsDailyFormattedQueryOptions = queryOptions({
    ...statsDailyQueryOptions,
    select: (data): StatsDailyFormattedResponse => ({
        max_request_count: data.max_request_count,
        items: data.items.map(formatStatsDaily),
    }),
    refetchInterval: 3600000, // 1 小时
    refetchOnMount: 'always',
});

/**
 * 获取每日统计数据 Hook
 */
export function useStatsDaily() {
    const query = useQuery(statsDailyFormattedQueryOptions);
    return {
        ...query,
        data: query.data?.items,
        maxRequestCount: query.data?.max_request_count ?? 0,
    };
}

// statsTodayFormattedQueryOptions 保留今日内存聚合，避免每日数据库快照出现短暂滞后。
const statsTodayFormattedQueryOptions = queryOptions({
    ...statsTodayQueryOptions,
    select: formatStatsDaily,
    refetchInterval: 10000,
    refetchOnMount: 'always',
});

export function useStatsToday() {
    return useQuery(statsTodayFormattedQueryOptions);
}

// statsHourlyFormattedQueryOptions 统一首页每小时统计查询、格式化和刷新策略。
const statsHourlyFormattedQueryOptions = queryOptions({
    ...statsHourlyQueryOptions,
    select: (data) => data.map((item): StatsHourlyFormatted => ({
        hour: item.hour,
        date: item.date,
        input_token: formatCount(item.input_token),
        output_token: formatCount(item.output_token),
        cache_read_token: formatCount(item.cache_read_token),
        cache_write_token: formatCount(item.cache_write_token),
        cache_hit_rate: formatCacheHitRate(item.input_token, item.cache_read_token),
        total_token: formatCount(item.input_token + item.output_token),
        input_cost: formatMoney(item.input_cost),
        output_cost: formatMoney(item.output_cost),
        total_cost: formatMoney(item.input_cost + item.output_cost),
        wait_time: formatTime(item.wait_time),
        request_success: formatCount(item.request_success),
        request_failed: formatCount(item.request_failed),
        request_count: formatCount(item.request_success + item.request_failed),
    })),
    refetchInterval: 10000,// 10 秒
    refetchOnMount: 'always',
});

/**
 * 获取每小时统计数据 Hook
 */
export function useStatsHourly() {
    return useQuery(statsHourlyFormattedQueryOptions);
}

// statsTotalFormattedQueryOptions 统一首页总统计查询、格式化和刷新策略。
const statsTotalFormattedQueryOptions = queryOptions({
    ...statsTotalQueryOptions,
    select: (data): StatsTotalFormatted => ({
        input_token: formatCount(data.input_token),
        output_token: formatCount(data.output_token),
        cache_read_token: formatCount(data.cache_read_token),
        cache_write_token: formatCount(data.cache_write_token),
        cache_hit_rate: formatCacheHitRate(data.input_token, data.cache_read_token),
        total_token: formatCount(data.input_token + data.output_token),
        input_cost: formatMoney(data.input_cost),
        output_cost: formatMoney(data.output_cost),
        total_cost: formatMoney(data.input_cost + data.output_cost),
        wait_time: formatTime(data.wait_time),
        request_success: formatCount(data.request_success),
        request_failed: formatCount(data.request_failed),
        request_count: formatCount(data.request_success + data.request_failed),
    }),
    refetchInterval: 10000,// 10 秒
    refetchOnMount: 'always',
});

/**
 * 获取总统计数据 Hook
 */
export function useStatsTotal() {
    return useQuery(statsTotalFormattedQueryOptions);
}

// useClearStats 清空服务端累计统计，并刷新首页与排行相关查询。
export function useClearStats() {
    const queryClient = useQueryClient();
    return useMutation({
        mutationFn: () => apiRequest<null>('/api/v1/stats/clear', { method: 'DELETE' }),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ['stats'] });
            queryClient.invalidateQueries({ queryKey: ['channels'] });
            queryClient.invalidateQueries({ queryKey: ['groups'] });
            queryClient.invalidateQueries({ queryKey: ['apikey'] });
        },
    });
}

export function useStatsSummary(period: StatsSummary['period']) {
    return useQuery({
        ...statsSummaryQueryOptions(period),
        select: (data): StatsSummaryFormatted => ({
            input_token: formatCount(data.input_token),
            output_token: formatCount(data.output_token),
            cache_read_token: formatCount(data.cache_read_token),
            cache_write_token: formatCount(data.cache_write_token),
            cache_hit_rate: formatCacheHitRate(data.input_token, data.cache_read_token),
            total_token: formatCount(data.input_token + data.output_token),
            input_cost: formatMoney(data.input_cost),
            output_cost: formatMoney(data.output_cost),
            total_cost: formatMoney(data.input_cost + data.output_cost),
            wait_time: formatTime(data.wait_time),
            request_success: formatCount(data.request_success),
            request_failed: formatCount(data.request_failed),
            request_count: formatCount(data.request_success + data.request_failed),
            period: data.period,
            points: data.points,
        }),
        refetchInterval: 10000,
        refetchOnMount: 'always',
    });
}



/**
 * 获取 API Key 统计数据列表 Hook
 */
export function useStatsAPIKey() {
    return useQuery({
        queryKey: ['stats', 'apikey'],
        queryFn: () => apiRequest<StatsAPIKey[]>('/api/v1/stats/apikey'),
        select: (data) => data.map((item): StatsAPIKeyFormatted => ({
            api_key_id: item.api_key_id,
            input_token: formatCount(item.input_token),
            output_token: formatCount(item.output_token),
            cache_read_token: formatCount(item.cache_read_token),
            cache_write_token: formatCount(item.cache_write_token),
            cache_hit_rate: formatCacheHitRate(item.input_token, item.cache_read_token),
            total_token: formatCount(item.input_token + item.output_token),
            input_cost: formatMoney(item.input_cost),
            output_cost: formatMoney(item.output_cost),
            total_cost: formatMoney(item.input_cost + item.output_cost),
            wait_time: formatTime(item.wait_time),
            request_success: formatCount(item.request_success),
            request_failed: formatCount(item.request_failed),
            request_count: formatCount(item.request_success + item.request_failed),
        })),
        refetchInterval: 30000,
        refetchOnMount: 'always',
    });
}
