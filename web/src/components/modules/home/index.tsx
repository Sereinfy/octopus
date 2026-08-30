import { useEffect, useRef, useState } from 'react';
import { Copy, Download, Eye, EyeOff, Loader2, Share2, X } from 'lucide-react';
import { snapdom } from '@zumer/snapdom';
import { toast } from 'sonner';
import dayjs from 'dayjs';
import { useTranslations } from 'use-intl';
import { buttonVariants } from '@/components/ui/button';
import Logo from '@/components/modules/logo';
import { StatsChart } from './chart';
import { Rank } from './rank';
import { useHomeViewStore } from './store';

// HomeSections 汇总首页正文, 屏内展示与分享图副本共用。
function HomeSections() {
    return (
        <div className="space-y-6">
            <StatsChart />
            <Rank />
        </div>
    );
}

// Home 渲染首页统计正文。
export function Home() {
    return (
        <div className="h-full min-h-0 overflow-y-auto overscroll-contain rounded-t-3xl pb-24 md:pb-4">
            <HomeSections />
        </div>
    );
}

// HomeActions 向稳定顶栏提供渠道名模糊开关和首页统计分享入口。
export function HomeActions() {
    const t = useTranslations('toolbar');
    const tCommon = useTranslations('common');
    const isChannelNameHidden = useHomeViewStore((state) => state.isChannelNameHidden);
    const setChannelNameHidden = useHomeViewStore((state) => state.setChannelNameHidden);
    const [isStaged, setIsStaged] = useState(false);
    const [preview, setPreview] = useState<{ url: string; blob: Blob } | null>(null);
    const stageRef = useRef<HTMLDivElement>(null);

    useEffect(() => () => {
        if (preview) URL.revokeObjectURL(preview.url);
    }, [preview]);

    useEffect(() => {
        if (!isStaged) return;

        let cancelled = false;
        const timer = window.setTimeout(async () => {
            if (cancelled || !stageRef.current) return;
            try {
                const blob = await snapdom.toBlob(stageRef.current, {
                    type: 'png',
                    scale: 2,
                    embedFonts: true,
                });
                if (!cancelled) setPreview({ url: URL.createObjectURL(blob), blob });
            } catch (error) {
                if (!cancelled) toast.error(error instanceof Error ? error.message : String(error));
            } finally {
                if (!cancelled) setIsStaged(false);
            }
        }, 1800);

        return () => {
            cancelled = true;
            window.clearTimeout(timer);
        };
    }, [isStaged]);

    const handleCopyImage = async () => {
        if (!preview) return;
        try {
            await navigator.clipboard.write([new ClipboardItem({ 'image/png': preview.blob })]);
            toast.success(tCommon('copy.success'));
        } catch {
            toast.error(tCommon('copy.failed'));
        }
    };

    const handleDownloadImage = () => {
        if (!preview) return;
        const link = document.createElement('a');
        link.href = preview.url;
        link.download = `octopus-${dayjs().format('YYYYMMDD')}.png`;
        link.click();
    };

    const iconButtonClass = buttonVariants({
        variant: 'ghost',
        size: 'icon',
        className: 'rounded-xl transition-none hover:bg-transparent text-muted-foreground hover:text-foreground disabled:opacity-50',
    });
    const previewActionClass = 'flex size-10 items-center justify-center rounded-xl border border-border bg-card text-muted-foreground transition-colors hover:text-foreground active:scale-95';

    return (
        <div className="flex items-center gap-1">
            <button
                type="button"
                onClick={() => setChannelNameHidden(!isChannelNameHidden)}
                aria-pressed={isChannelNameHidden}
                aria-label={t('hideChannelName')}
                title={t('hideChannelName')}
                className={iconButtonClass}
            >
                {isChannelNameHidden ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
            </button>

            <button
                type="button"
                onClick={() => setIsStaged(true)}
                disabled={isStaged}
                aria-label={t('share')}
                title={t('share')}
                className={iconButtonClass}
            >
                {isStaged ? <Loader2 className="size-4 animate-spin" /> : <Share2 className="size-4" />}
            </button>

            {isStaged && (
                <div
                    ref={stageRef}
                    aria-hidden="true"
                    className="fixed top-0 rounded-3xl bg-background p-4 text-foreground"
                    style={{ left: '-10000px', width: '768px' }}
                >
                    <div className="mb-4 flex items-center gap-x-2 px-2">
                        <Logo size={48} />
                        <span className="text-3xl font-bold">Octopus</span>
                    </div>
                    <HomeSections />
                </div>
            )}

            {preview && (
                <div className="fixed inset-0 z-60 flex flex-col items-center justify-center gap-3 bg-background/95 p-4 backdrop-blur-sm">
                    <img
                        src={preview.url}
                        alt=""
                        className="max-h-[75vh] min-h-0 w-auto max-w-full rounded-2xl border border-border object-contain"
                    />
                    <div className="flex shrink-0 items-center gap-2">
                        <button type="button" onClick={handleCopyImage} aria-label={tCommon('copy.success')} title={tCommon('copy.success')} className={previewActionClass}>
                            <Copy className="size-4" />
                        </button>
                        <button type="button" onClick={handleDownloadImage} aria-label={t('download')} title={t('download')} className={previewActionClass}>
                            <Download className="size-4" />
                        </button>
                        <button type="button" onClick={() => setPreview(null)} aria-label={t('close')} title={t('close')} className={previewActionClass}>
                            <X className="size-4" />
                        </button>
                    </div>
                </div>
            )}
        </div>
    );
}
