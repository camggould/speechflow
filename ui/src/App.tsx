import { useEffect } from "react";
import { HeroUIProvider } from "@heroui/react";
import { QueryClientProvider } from "@tanstack/react-query";
import { Route, Router, Switch } from "wouter";
import { queryClient } from "@/api/query";
import { useAppStore } from "@/store/app";
import { Layout } from "@/components/Layout";
import { ErrorBoundary } from "@/components/ErrorBoundary";
import { Dashboard } from "@/views/Dashboard";
import { SessionView } from "@/views/SessionView";
import { IterationView } from "@/views/IterationView";
import { ROUTE_PATTERNS } from "@/routes";
import "@/styles/globals.css";

function NotFound() {
  return (
    <div className="p-8 text-sm text-default-500">
      Not found. <a href="/ui/">Back to dashboard</a>
    </div>
  );
}

function AppRoutes() {
  return (
    <Layout>
      <ErrorBoundary>
        <Switch>
          <Route path={ROUTE_PATTERNS.dashboard} component={Dashboard} />
          <Route path={ROUTE_PATTERNS.iteration}>
            {(params) => (
              <IterationView
                sessionId={params.sessionId}
                iterationId={params.iterationId}
              />
            )}
          </Route>
          <Route path={ROUTE_PATTERNS.session}>
            {(params) => <SessionView sessionId={params.sessionId} />}
          </Route>
          <Route component={NotFound} />
        </Switch>
      </ErrorBoundary>
    </Layout>
  );
}

function ThemeWrapper({ children }: { children: React.ReactNode }) {
  const theme = useAppStore((s) => s.theme);
  const resolved =
    theme === "system"
      ? window.matchMedia("(prefers-color-scheme: dark)").matches
        ? "dark"
        : "light"
      : theme;

  // Set the class on <html> so HeroUI Modals (portaled to document.body)
  // and ReactFlow's CSS variables see it. Local div class isn't enough
  // because portals escape the React tree.
  useEffect(() => {
    const root = document.documentElement;
    root.classList.remove("light", "dark");
    root.classList.add(resolved);
    root.dataset.theme = resolved;
    root.style.colorScheme = resolved;
  }, [resolved]);

  return (
    <div className="text-foreground bg-background min-h-screen">{children}</div>
  );
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <HeroUIProvider>
        <ThemeWrapper>
          <Router base="/ui">
            <AppRoutes />
          </Router>
        </ThemeWrapper>
      </HeroUIProvider>
    </QueryClientProvider>
  );
}
