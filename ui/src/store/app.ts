import { create } from "zustand";
import { persist } from "zustand/middleware";

export type Theme = "light" | "dark" | "system";
export type PlaybackMode = "live" | "playback";
export type PlaybackSpeed = 0.5 | 1 | 2 | 4;

export interface PlaybackState {
  mode: PlaybackMode;
  // ISO 8601 string. In live mode this is effectively "now"; in playback
  // mode it's the scrubber position relative to the iteration's timeline.
  cursor: string;
  speed: PlaybackSpeed;
  playing: boolean;
}

interface AppState {
  theme: Theme;
  setTheme: (t: Theme) => void;

  playback: PlaybackState;
  setMode: (mode: PlaybackMode) => void;
  setCursor: (cursor: string) => void;
  setSpeed: (speed: PlaybackSpeed) => void;
  setPlaying: (playing: boolean) => void;

  transcriptOpen: boolean;
  openTranscript: () => void;
  closeTranscript: () => void;

  focusedNodeId: string | null;
  focusNode: (id: string | null) => void;

  // Toggle for the session-view right pane: iteration view (default) vs
  // the cross-iteration coverage matrix.
  coverageMatrixOpen: boolean;
  setCoverageMatrixOpen: (open: boolean) => void;
}

export const useAppStore = create<AppState>()(
  persist(
    (set) => ({
      theme: "system",
      setTheme: (theme) => set({ theme }),

      playback: {
        mode: "live",
        cursor: new Date().toISOString(),
        speed: 1,
        playing: false,
      },
      setMode: (mode) =>
        set((s) => ({ playback: { ...s.playback, mode } })),
      setCursor: (cursor) =>
        set((s) => ({ playback: { ...s.playback, cursor } })),
      setSpeed: (speed) =>
        set((s) => ({ playback: { ...s.playback, speed } })),
      setPlaying: (playing) =>
        set((s) => ({ playback: { ...s.playback, playing } })),

      transcriptOpen: false,
      openTranscript: () => set({ transcriptOpen: true }),
      closeTranscript: () => set({ transcriptOpen: false }),

      focusedNodeId: null,
      focusNode: (id) => set({ focusedNodeId: id }),

      coverageMatrixOpen: false,
      setCoverageMatrixOpen: (open) => set({ coverageMatrixOpen: open }),
    }),
    {
      name: "speechflow-ui",
      partialize: (s) => ({ theme: s.theme }),
    },
  ),
);
