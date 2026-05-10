export const ROUTES = {
  dashboard: "/",
  session: (sessionId: string) => `/sessions/${sessionId}`,
  iteration: (sessionId: string, iterationId: string) =>
    `/sessions/${sessionId}/iterations/${iterationId}`,
} as const;

export const ROUTE_PATTERNS = {
  dashboard: "/",
  session: "/sessions/:sessionId",
  iteration: "/sessions/:sessionId/iterations/:iterationId",
} as const;
