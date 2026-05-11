import { Tab, Tabs } from "@heroui/react";
import { CoveragePanel } from "@/components/CoveragePanel";
import { SpeechHealthPanel } from "@/components/SpeechHealthPanel";

interface InsightsPanelProps {
  iterationId: string;
}

// Right-side tabbed panel. "Coverage" answers "did I touch the topics I
// declared?" — structural and binary. "Health" surfaces the rhetorical
// quality the agent annotated during the iteration (strengths,
// weaknesses, airtime balance) — diagnostic and quantitative. Both run
// against the same iteration; the user toggles.
export function InsightsPanel({ iterationId }: InsightsPanelProps) {
  return (
    <div className="h-full w-80 border-l border-divider flex flex-col">
      <Tabs
        aria-label="Iteration insights"
        size="sm"
        variant="underlined"
        classNames={{
          base: "px-3 pt-2",
          tabList: "gap-4",
          panel: "flex-1 min-h-0 p-0",
        }}
      >
        <Tab key="coverage" title="Coverage">
          <CoveragePanel iterationId={iterationId} embedded />
        </Tab>
        <Tab key="health" title="Health">
          <SpeechHealthPanel iterationId={iterationId} />
        </Tab>
      </Tabs>
    </div>
  );
}
