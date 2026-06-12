import { useEffect } from "react";
import { Folder, MoreHorizontal, Plus, Search, Star } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { WorkspaceSwitcher } from "@/components/workspace-switcher";
import { AdminLogin } from "@/components/admin-login";
import { FolderBreadcrumbs } from "@/components/folder-breadcrumbs";
import { FolderList } from "@/components/folder-list";
import { FolderTree } from "@/components/folder-tree";
import useAuthStore from "./store/auth-store";
import useFolderStore from "./store/folder-store";
import useWorkspaceStore from "./store/workspace-store";

const recentScenes = [
  { title: "Launch flow", folder: "Product Boards", updated: "Today" },
  { title: "Auth rewrite", folder: "Engineering", updated: "Yesterday" },
  { title: "Dashboard IA", folder: "Design Ops", updated: "Jun 3" },
];

function App() {
  const checkAuth = useAuthStore((state) => state.checkAuth);
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const isLoading = useAuthStore((state) => state.isLoading);
  const { workspaces, activeWorkspaceId } = useWorkspaceStore();
  const fetchFolders = useFolderStore((state) => state.fetchFolders);
  const resetFolders = useFolderStore((state) => state.reset);

  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  useEffect(() => {
    if (!activeWorkspaceId) {
      resetFolders();
      return;
    }
    resetFolders();
    void fetchFolders(activeWorkspaceId).catch(() => undefined);
  }, [activeWorkspaceId, fetchFolders, resetFolders]);

  if (isLoading) {
    return <main className="grid min-h-screen place-items-center">Loading...</main>;
  }

  if (!isAuthenticated) {
    return <AdminLogin />;
  }

  const activeWorkspace = Array.isArray(workspaces) ? workspaces.find((w) => w.id === activeWorkspaceId) : null;

  return (
    <main className="grid min-h-screen grid-cols-[248px_minmax(0,1fr)] bg-background max-md:grid-cols-1">
      <aside
        className="flex flex-col gap-6 border-r bg-card px-4 py-6 max-md:border-b max-md:border-r-0"
        aria-label="Workspace navigation"
      >
        <WorkspaceSwitcher />

        <div className="max-h-[calc(100vh-7rem)] overflow-y-auto pr-2 max-md:max-h-none">
          <FolderTree />
        </div>
      </aside>

      <section className="grid content-start gap-7 p-7">
        <FolderBreadcrumbs />
        <header className="flex items-center justify-between gap-5 max-md:flex-col max-md:items-stretch">
          <div>
            <p className="mb-1 text-xs font-bold uppercase text-muted-foreground">Workspace</p>
            <h1 className="text-3xl font-semibold leading-tight">{activeWorkspace?.name || "Select Workspace"}</h1>
          </div>
...

          <div className="flex items-center gap-3 max-md:flex-col max-md:items-stretch">
            <label className="relative block w-80 max-w-full">
              <Search
                className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground"
                size={16}
                aria-hidden="true"
              />
              <Input aria-label="Search scenes" className="pl-9" placeholder="Search scenes" />
            </label>
            <Button type="button">
              <Plus size={16} aria-hidden="true" />
              <span>New scene</span>
            </Button>
          </div>
        </header>

        <FolderList />

        <section className="grid gap-4" aria-label="Recent scenes">
          <h2 className="text-lg font-semibold">Recent scenes</h2>
        <div className="grid grid-cols-[repeat(auto-fit,minmax(220px,1fr))] gap-4">
          {recentScenes.map((scene) => (
            <Card key={scene.title}>
              <CardHeader className="space-y-4">
                <div className="flex items-center justify-between text-muted-foreground">
                  <Folder size={18} aria-hidden="true" />
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button aria-label="Scene actions" size="icon" type="button" variant="ghost">
                        <MoreHorizontal size={16} aria-hidden="true" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem>Open</DropdownMenuItem>
                      <DropdownMenuItem>
                        <Star size={16} aria-hidden="true" />
                        Star
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
                <CardTitle className="text-lg">{scene.title}</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-2 text-sm text-muted-foreground">
                <p>{scene.folder}</p>
                <span>{scene.updated}</span>
              </CardContent>
            </Card>
          ))}
        </div>
        </section>
      </section>
    </main>
  );
}

export default App;
