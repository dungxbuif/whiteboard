import { PanelLeft } from "lucide-react";
import { useState, type ReactNode } from "react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogTitle, DialogTrigger } from "@/components/ui/dialog";
import { FolderBreadcrumbs } from "@/components/folder-breadcrumbs";
import { SurfaceThemeContext, type SurfaceTheme } from "@/lib/surface-theme";
import { DashboardSidebar } from "./dashboard-sidebar";
import "@/styles/dashboard.css";

const THEME_KEY = "dashboard-theme";
const COLLAPSE_KEY = "dashboard-sidebar-collapsed";
function readPreference(key: string) { try { return localStorage.getItem(key); } catch { return null; } }
function savePreference(key: string, value: string) { try { localStorage.setItem(key, value); } catch { /* Browser storage can be disabled. */ } }

export function DashboardShell({ children }: { children: ReactNode }) {
  const [theme, setTheme] = useState<SurfaceTheme>(() => readPreference(THEME_KEY) === "dark" ? "dark" : "light");
  const [collapsed, setCollapsed] = useState(() => readPreference(COLLAPSE_KEY) === "true");
  const [mobileOpen, setMobileOpen] = useState(false);
  const onThemeChange = (value: SurfaceTheme) => { setTheme(value); savePreference(THEME_KEY, value); };
  return <SurfaceThemeContext.Provider value={theme}>
    <main className="dashboard-theme flex h-dvh overflow-hidden bg-background text-foreground" data-theme={theme}>
      <aside className={`shrink-0 border-r max-md:hidden ${collapsed ? "hidden" : "block"}`}><DashboardSidebar theme={theme} onThemeChange={onThemeChange} /></aside>
      <div className="flex min-w-0 flex-1 flex-col">
        <div className="flex h-14 shrink-0 items-center gap-3 border-b px-4">
          <Button variant="ghost" size="icon" className="max-md:hidden" aria-label={collapsed ? "Expand sidebar" : "Collapse sidebar"} aria-expanded={!collapsed} onClick={() => { setCollapsed(!collapsed); savePreference(COLLAPSE_KEY, String(!collapsed)); }}><PanelLeft /></Button>
          <Dialog open={mobileOpen} onOpenChange={setMobileOpen}>
            <DialogTrigger asChild><Button variant="ghost" size="icon" className="md:hidden" aria-label="Open navigation"><PanelLeft /></Button></DialogTrigger>
            <DialogContent className="dashboard-drawer" aria-describedby={undefined}>
              <DialogTitle className="sr-only">Workspace navigation</DialogTitle>
              <DashboardSidebar theme={theme} onThemeChange={onThemeChange} onNavigate={() => setMobileOpen(false)} />
            </DialogContent>
          </Dialog>
          <span className="h-4 w-px bg-border" aria-hidden="true" />
          <div className="min-w-0 overflow-x-auto"><FolderBreadcrumbs /></div>
        </div>
        <div className="min-h-0 flex-1 overflow-y-auto p-8 max-sm:p-4"><div className="mx-auto grid max-w-screen-2xl content-start gap-8">{children}</div></div>
      </div>
    </main>
  </SurfaceThemeContext.Provider>;
}
