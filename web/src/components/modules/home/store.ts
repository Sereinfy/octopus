import { create } from 'zustand';
import { createJSONStorage, persist } from 'zustand/middleware';

export type RankSortMode = 'cost' | 'count' | 'tokens';
export type RankScope = 'channel' | 'model';
export type ChartPeriod = '1' | '7' | '30' | 'all';

interface HomeViewState {
    rankSortMode: RankSortMode;
    rankScope: RankScope;
    chartPeriod: ChartPeriod;
    setRankSortMode: (value: RankSortMode) => void;
    setRankScope: (value: RankScope) => void;
    setChartPeriod: (value: ChartPeriod) => void;
}

export const useHomeViewStore = create<HomeViewState>()(
    persist(
        (set) => ({
            rankSortMode: 'cost',
            rankScope: 'channel',
            chartPeriod: '7',
            setRankSortMode: (value) => set({ rankSortMode: value }),
            setRankScope: (value) => set({ rankScope: value }),
            setChartPeriod: (value) => set({ chartPeriod: value }),
        }),
        {
            name: 'home-view-options-storage',
            storage: createJSONStorage(() => localStorage),
            partialize: (state) => ({
                rankSortMode: state.rankSortMode,
                rankScope: state.rankScope,
                chartPeriod: state.chartPeriod,
            }),
        }
    )
);
