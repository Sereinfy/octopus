import { useEffect, type ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { motion } from 'motion/react';

// API Key 浮层挂载到 body，避免被设置页的滚动容器或多栏布局裁切。
export function OverlayPortal({ onClose, children }: { onClose: () => void; children: ReactNode }) {
    useEffect(() => {
        const handleKeyDown = (event: KeyboardEvent) => {
            if (event.key === 'Escape' && !event.defaultPrevented) {
                onClose();
            }
        };

        document.addEventListener('keydown', handleKeyDown);
        return () => document.removeEventListener('keydown', handleKeyDown);
    }, [onClose]);

    return createPortal(
        <>
            <motion.div
                data-slot="dialog-overlay"
                aria-hidden="true"
                className="fixed inset-0 z-50 bg-white/40 backdrop-blur-xs dark:bg-black/40"
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                onClick={onClose}
            />
            {children}
        </>,
        document.body
    );
}
