import { BarChart3, ScrollText, Trash2 } from 'lucide-react';
import { useState } from 'react';
import { useTranslations } from 'use-intl';
import { useClearLogs } from '@/api/log';
import { useClearStats } from '@/api/stats';
import { Button } from '@/components/ui/button';
import { toast } from 'sonner';

// SettingLog 提供进程内完成日志的清空操作。
export function SettingLog() {
    const t = useTranslations('setting');
    const clearLogs = useClearLogs();
    const clearStats = useClearStats();
    const [confirmClearStats, setConfirmClearStats] = useState(false);

    const handleClearLogs = () => {
        clearLogs.mutate(undefined, {
            onSuccess: () => toast.success(t('log.clearSuccess')),
            onError: () => toast.error(t('log.clearFailed')),
        });
    };

    const handleClearStats = () => {
        clearStats.mutate(undefined, {
            onSuccess: () => {
                setConfirmClearStats(false);
                toast.success(t('log.clearStatsSuccess'));
            },
            onError: () => toast.error(t('log.clearStatsFailed')),
        });
    };

    return (
        <div className="space-y-5 rounded-3xl border border-border bg-card p-6">
            <h2 className="flex items-center gap-2 text-lg font-bold text-card-foreground">
                <ScrollText className="size-5" />
                {t('log.title')}
            </h2>
            <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                    <Trash2 className="size-5 text-muted-foreground" />
                    <span className="text-sm font-medium">{t('log.clear.label')}</span>
                </div>
                <Button
                    variant="destructive"
                    size="sm"
                    onClick={handleClearLogs}
                    disabled={clearLogs.isPending}
                    className="rounded-xl"
                >
                    {clearLogs.isPending ? t('log.clear.clearing') : t('log.clear.button')}
                </Button>
            </div>
            <div className="flex items-center justify-between gap-4 border-t border-border/60 pt-5">
                <div className="flex items-center gap-3">
                    <BarChart3 className="size-5 text-muted-foreground" />
                    <span className="text-sm font-medium">{t('log.clearStats.label')}</span>
                </div>
                {!confirmClearStats ? (
                    <Button
                        variant="destructive"
                        size="sm"
                        onClick={() => setConfirmClearStats(true)}
                        disabled={clearStats.isPending}
                        className="rounded-xl"
                    >
                        {t('log.clearStats.button')}
                    </Button>
                ) : (
                    <div className="flex items-center gap-2">
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={() => setConfirmClearStats(false)}
                            disabled={clearStats.isPending}
                            className="rounded-xl"
                        >
                            {t('log.clearStats.cancel')}
                        </Button>
                        <Button
                            variant="destructive"
                            size="sm"
                            onClick={handleClearStats}
                            disabled={clearStats.isPending}
                            className="rounded-xl"
                        >
                            {clearStats.isPending ? t('log.clearStats.clearing') : t('log.clearStats.confirm')}
                        </Button>
                    </div>
                )}
            </div>
        </div>
    );
}
