import { Loader2, Logs, Pause, Play } from 'lucide-react';
import { useTranslations } from 'use-intl';
import { useLogViewStore, useLogs } from '@/api/log';
import { Button } from '@/components/ui/button';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { VirtualizedGrid } from '@/components/common/VirtualizedGrid';
import { LogCard } from './Item';

// LogActions exposes the log stream pause/resume control in the shared top-right action area.
export function LogActions() {
    const t = useTranslations('log.actions');
    const paused = useLogViewStore((state) => state.paused);
    const togglePaused = useLogViewStore((state) => state.togglePaused);
    const label = paused ? t('resume') : t('pause');

    return (
        <Tooltip>
            <TooltipTrigger asChild>
                <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    aria-label={label}
                    aria-pressed={paused}
                    onClick={togglePaused}
                    className="rounded-xl text-muted-foreground hover:text-foreground"
                >
                    {paused ? <Play className="size-4" /> : <Pause className="size-4" />}
                </Button>
            </TooltipTrigger>
            <TooltipContent side="bottom">{label}</TooltipContent>
        </Tooltip>
    );
}

// Log 展示进程内日志概览，并按 RequestID 实时更新卡片。
export function Log() {
    const t = useTranslations('log');
    const { logs, isLoading, error } = useLogs();

    if (isLoading) {
        return (
            <div className="flex h-full items-center justify-center">
                <Loader2 className="size-6 animate-spin text-muted-foreground" />
            </div>
        );
    }

    if (logs.length === 0) {
        return (
            <div className="flex h-full flex-col items-center justify-center gap-3 text-muted-foreground">
                {!error && <Logs className="size-8" />}
                <span className="text-sm">{error ? t('list.disconnected') : t('list.empty')}</span>
            </div>
        );
    }

    return (
        <div className="flex h-full min-h-0 flex-col gap-3">
            {error && (
                <div className="flex shrink-0 items-center justify-center px-1 pb-3 text-xs text-destructive">
                    <span>{t('list.disconnected')}</span>
                </div>
            )}
            <div className="min-h-0 flex-1">
                <VirtualizedGrid
                    items={logs}
                    layout="list"
                    columns={{ default: 1 }}
                    estimateItemHeight={104}
                    overscan={8}
                    getItemKey={(log) => `log-${log.id}`}
                    renderItem={(log) => <LogCard log={log} />}
                />
            </div>
        </div>
    );
}
