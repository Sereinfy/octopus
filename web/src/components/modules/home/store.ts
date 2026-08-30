import { create } from 'zustand';
import { createJSONStorage, persist } from 'zustand/middleware';

export type RankSortMode = 'cost' | 'count' | 'tokens';
export type ChartPeriod = '1' | '7' | '30' | 'all';

interface HomeViewState {
    channelRankSortMode: RankSortMode;
    modelRankSortMode: RankSortMode;
    chartPeriod: ChartPeriod;
    isChannelNameHidden: boolean;
    setChannelRankSortMode: (value: RankSortMode) => void;
    setModelRankSortMode: (value: RankSortMode) => void;
    setChartPeriod: (value: ChartPeriod) => void;
    setChannelNameHidden: (value: boolean) => void;
}

export const useHomeViewStore = create<HomeViewState>()(
    persist(
        (set) => ({
            channelRankSortMode: 'cost',
            modelRankSortMode: 'cost',
            chartPeriod: '7',
            isChannelNameHidden: false,
            setChannelRankSortMode: (value) => set({ channelRankSortMode: value }),
            setModelRankSortMode: (value) => set({ modelRankSortMode: value }),
            setChartPeriod: (value) => set({ chartPeriod: value }),
            setChannelNameHidden: (value) => set({ isChannelNameHidden: value }),
        }),
        {
            name: 'home-view-options-storage',
            storage: createJSONStorage(() => localStorage),
            merge: (persistedState, currentState) => {
                const persisted = persistedState as Partial<HomeViewState> & { rankSortMode?: RankSortMode };
                const legacySortMode = persisted.rankSortMode;
                return {
                    ...currentState,
                    ...persisted,
                    channelRankSortMode: persisted.channelRankSortMode ?? legacySortMode ?? currentState.channelRankSortMode,
                    modelRankSortMode: persisted.modelRankSortMode ?? legacySortMode ?? currentState.modelRankSortMode,
                };
            },
            partialize: (state) => ({
                channelRankSortMode: state.channelRankSortMode,
                modelRankSortMode: state.modelRankSortMode,
                chartPeriod: state.chartPeriod,
            }),
        }
    )
);
