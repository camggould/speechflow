import { Link } from "wouter";
import { Card, CardBody, CardHeader } from "@heroui/react";
import { formatDistanceToNow } from "date-fns";
import { Sparkles } from "lucide-react";
import { useSessions } from "@/api/query";
import { SessionCard } from "@/components/SessionCard";
import { ROUTES } from "@/routes";

export function Dashboard() {
  const { data, isLoading, error } = useSessions();
  const sessions = data ?? [];

  // Recent activity: top 10 by last_activity_at desc.
  const recent = [...sessions]
    .sort(
      (a, b) =>
        new Date(b.last_activity_at).getTime() -
        new Date(a.last_activity_at).getTime(),
    )
    .slice(0, 10);

  return (
    <div className="p-6 h-full overflow-auto">
      <div className="max-w-7xl mx-auto grid grid-cols-1 lg:grid-cols-[1fr_320px] gap-6">
        <section>
          <div className="flex items-center justify-between mb-4">
            <h1 className="text-2xl font-bold">Sessions</h1>
          </div>

          {isLoading && (
            <div className="text-sm text-default-500">Loading sessions…</div>
          )}

          {error && (
            <Card>
              <CardBody className="text-sm text-default-500">
                Couldn't reach the backend. Is{" "}
                <code>speechflow serve</code> running on{" "}
                <code>localhost:7777</code>?
              </CardBody>
            </Card>
          )}

          {!isLoading && !error && sessions.length === 0 && (
            <Card>
              <CardBody className="gap-3 py-8 items-center text-center">
                <Sparkles size={32} className="text-default-400" />
                <h2 className="text-lg font-semibold">No sessions yet</h2>
                <p className="text-sm text-default-500 max-w-md">
                  Sessions are created from the CLI. Run{" "}
                  <code>speechflow session new --title "My talk"</code> to get
                  started, then come back here.
                </p>
              </CardBody>
            </Card>
          )}

          {sessions.length > 0 && (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              {sessions.map((s) => (
                <SessionCard key={s.id} session={s} />
              ))}
            </div>
          )}
        </section>

        <aside>
          <Card>
            <CardHeader className="pb-2">
              <h2 className="text-sm font-semibold uppercase tracking-wider text-default-500">
                Recent activity
              </h2>
            </CardHeader>
            <CardBody className="gap-1 pt-0">
              {recent.length === 0 && (
                <span className="text-xs text-default-400">Nothing yet.</span>
              )}
              {recent.map((s) => (
                <Link
                  key={s.id}
                  href={ROUTES.session(s.id)}
                  className="block p-2 rounded hover:bg-default-100 dark:hover:bg-default-800/40"
                >
                  <div className="text-sm font-medium truncate">{s.title}</div>
                  <div className="text-[10px] text-default-400">
                    {formatDistanceToNow(new Date(s.last_activity_at), {
                      addSuffix: true,
                    })}{" "}
                    · {s.iteration_count} iteration
                    {s.iteration_count === 1 ? "" : "s"}
                  </div>
                </Link>
              ))}
            </CardBody>
          </Card>
        </aside>
      </div>
    </div>
  );
}
