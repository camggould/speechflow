import { useEffect, useRef } from "react";
import { Button, Slider } from "@heroui/react";
import { Pause, Play } from "lucide-react";
import { useAppStore, type PlaybackSpeed } from "@/store/app";

interface TimelineScrubberProps {
  startedAt: string;
  endedAt: string | null;
  // Latest event timestamp in the iteration (max of node/edge created_at).
  // Used to derive the scrub upper bound when the iteration is still active.
  latestEventAt: string | null;
}

const SPEEDS: PlaybackSpeed[] = [0.5, 1, 2, 4];
const MIN_WINDOW_MS = 5000;

export function TimelineScrubber({
  startedAt,
  endedAt,
  latestEventAt,
}: TimelineScrubberProps) {
  const cursor = useAppStore((s) => s.playback.cursor);
  const playing = useAppStore((s) => s.playback.playing);
  const speed = useAppStore((s) => s.playback.speed);
  const setCursor = useAppStore((s) => s.setCursor);
  const setPlaying = useAppStore((s) => s.setPlaying);
  const setSpeed = useAppStore((s) => s.setSpeed);

  const startMs = new Date(startedAt).getTime();
  // Upper bound: when the iteration has ended, use ended_at. Otherwise use
  // the latest event timestamp with a 1s tail so the playhead can settle at
  // the end. Always enforce a 5s minimum window so the slider is usable
  // even when the iteration is empty.
  const rawEndMs = endedAt
    ? new Date(endedAt).getTime()
    : latestEventAt
    ? new Date(latestEventAt).getTime() + 1000
    : startMs + MIN_WINDOW_MS;
  const endMs = Math.max(startMs + MIN_WINDOW_MS, rawEndMs);
  const cursorMs = Math.min(Math.max(new Date(cursor).getTime(), startMs), endMs);

  // requestAnimationFrame loop drives the cursor forward at `speed`. Reads
  // the live cursor from the store inside `tick` so we don't have to restart
  // RAF on every cursor mutation.
  const rafRef = useRef<number | null>(null);
  useEffect(() => {
    if (!playing) {
      if (rafRef.current != null) cancelAnimationFrame(rafRef.current);
      rafRef.current = null;
      return;
    }

    let lastTick = 0;
    const tick = (now: number) => {
      if (lastTick === 0) {
        lastTick = now;
        rafRef.current = requestAnimationFrame(tick);
        return;
      }
      const dt = now - lastTick;
      lastTick = now;
      const currentMs = new Date(useAppStore.getState().playback.cursor).getTime();
      const next = currentMs + dt * speed;
      if (next >= endMs) {
        setCursor(new Date(endMs).toISOString());
        setPlaying(false);
        return;
      }
      setCursor(new Date(next).toISOString());
      rafRef.current = requestAnimationFrame(tick);
    };
    rafRef.current = requestAnimationFrame(tick);
    return () => {
      if (rafRef.current != null) cancelAnimationFrame(rafRef.current);
    };
  }, [playing, speed, endMs, setCursor, setPlaying]);

  const handlePlay = () => {
    // If we're at the end, restart from the beginning on play.
    if (!playing && cursorMs >= endMs - 50) {
      setCursor(new Date(startMs).toISOString());
    }
    setPlaying(!playing);
  };

  const elapsedMs = cursorMs - startMs;
  const totalMs = endMs - startMs;

  return (
    <div className="flex items-center gap-3 flex-1 min-w-0">
      <Button
        isIconOnly
        variant="flat"
        size="sm"
        onPress={handlePlay}
        aria-label={playing ? "Pause" : "Play"}
      >
        {playing ? <Pause size={14} /> : <Play size={14} />}
      </Button>
      <Slider
        size="sm"
        minValue={startMs}
        maxValue={endMs}
        step={50}
        value={cursorMs}
        onChange={(val) => {
          const v = Array.isArray(val) ? val[0] : val;
          setCursor(new Date(v).toISOString());
        }}
        aria-label="Timeline"
        className="flex-1"
      />
      <div className="text-[10px] tabular-nums text-default-500 w-16 text-right">
        {(elapsedMs / 1000).toFixed(1)}s / {(totalMs / 1000).toFixed(1)}s
      </div>
      <div className="flex items-center gap-1">
        {SPEEDS.map((s) => (
          <Button
            key={s}
            variant={s === speed ? "solid" : "light"}
            size="sm"
            onPress={() => setSpeed(s)}
            className="min-w-10 px-2"
          >
            {s}x
          </Button>
        ))}
      </div>
    </div>
  );
}
