import { Link } from "wouter";
import {
  Navbar,
  NavbarBrand,
  NavbarContent,
  NavbarItem,
  Button,
} from "@heroui/react";
import { Mic, Sun, Moon, Monitor } from "lucide-react";
import { useAppStore } from "@/store/app";
import { ROUTES } from "@/routes";

interface LayoutProps {
  children: React.ReactNode;
}

export function Layout({ children }: LayoutProps) {
  const theme = useAppStore((s) => s.theme);
  const setTheme = useAppStore((s) => s.setTheme);

  const themeIcon =
    theme === "light" ? (
      <Sun size={16} />
    ) : theme === "dark" ? (
      <Moon size={16} />
    ) : (
      <Monitor size={16} />
    );

  const cycleTheme = () => {
    if (theme === "system") setTheme("light");
    else if (theme === "light") setTheme("dark");
    else setTheme("system");
  };

  return (
    <div className="min-h-screen flex flex-col">
      <Navbar isBordered maxWidth="full">
        <NavbarBrand>
          <Link href={ROUTES.dashboard} className="flex items-center gap-2 font-bold text-lg">
            <Mic size={20} className="text-primary" />
            speechflow
          </Link>
        </NavbarBrand>

        <NavbarContent justify="end" className="gap-1">
          <NavbarItem>
            <Button
              isIconOnly
              variant="light"
              size="sm"
              onPress={cycleTheme}
              aria-label={`Theme: ${theme}`}
            >
              {themeIcon}
            </Button>
          </NavbarItem>
        </NavbarContent>
      </Navbar>

      <main className="flex-1 overflow-hidden">{children}</main>
    </div>
  );
}
