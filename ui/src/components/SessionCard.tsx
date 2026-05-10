import { Link } from "wouter";
import { Card, CardBody, CardHeader } from "@heroui/react";
import { formatDistanceToNow } from "date-fns";
import type { Session } from "@/api/types.gen";
import { ROUTES } from "@/routes";

interface SessionCardProps {
  session: Session;
}

export function SessionCard({ session }: SessionCardProps) {
  const coverage =
    session.latest_coverage_pct != null
      ? `${Math.round(session.latest_coverage_pct * 100)}%`
      : "—";

  return (
    <Link href={ROUTES.session(session.id)}>
      <Card isPressable isHoverable className="w-full text-left">
        <CardHeader className="flex flex-col items-start gap-1 pb-1">
          <h3 className="text-base font-semibold line-clamp-1">{session.title}</h3>
          {session.description && (
            <p className="text-xs text-default-500 line-clamp-2">
              {session.description}
            </p>
          )}
        </CardHeader>
        <CardBody className="pt-1 flex flex-row items-end justify-between gap-2">
          <div className="flex flex-col gap-0.5">
            <span className="text-[10px] uppercase tracking-wider text-default-400">
              Iterations
            </span>
            <span className="text-sm font-medium">{session.iteration_count}</span>
          </div>
          <div className="flex flex-col gap-0.5">
            <span className="text-[10px] uppercase tracking-wider text-default-400">
              Coverage
            </span>
            <span className="text-sm font-medium">{coverage}</span>
          </div>
          <div className="flex flex-col gap-0.5 items-end">
            <span className="text-[10px] uppercase tracking-wider text-default-400">
              Last activity
            </span>
            <span className="text-xs text-default-500">
              {formatDistanceToNow(new Date(session.last_activity_at), {
                addSuffix: true,
              })}
            </span>
          </div>
        </CardBody>
      </Card>
    </Link>
  );
}
