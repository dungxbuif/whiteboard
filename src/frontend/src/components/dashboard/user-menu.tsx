import { ChevronsUpDown, LogOut, Moon, Sun } from "lucide-react";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuLabel, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import type { SurfaceTheme } from "@/lib/surface-theme";
import useAuthStore from "@/store/auth-store";
import useSceneStore from "@/store/scene-store";
import useFolderStore from "@/store/folder-store";
import useSearchStore from "@/store/search-store";
import useWorkspaceStore from "@/store/workspace-store";
import { navigate } from "@/lib/router";

export function UserMenu({ theme, onThemeChange }: { theme: SurfaceTheme; onThemeChange: (value: SurfaceTheme) => void }) {
  const user = useAuthStore(state => state.user);
  const label = user?.name || user?.email || "Account";
  return <DropdownMenu>
    <DropdownMenuTrigger asChild>
      <Button variant="ghost" aria-label="Account menu" className="h-auto w-full justify-start gap-2 px-2 py-2 text-left">
        <span className="grid size-8 shrink-0 place-items-center rounded-lg bg-muted text-xs">{label.slice(0, 2).toUpperCase()}</span>
        <span className="grid min-w-0 flex-1 text-sm"><span className="truncate font-semibold">{label}</span><span className="truncate text-xs font-normal text-muted-foreground">{user?.email}</span></span>
        <ChevronsUpDown className="size-4 shrink-0" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent align="end" side="top" className="w-60">
      <DropdownMenuLabel className="truncate">{user?.email}</DropdownMenuLabel>
      <DropdownMenuSeparator />
      <DropdownMenuItem onSelect={() => onThemeChange(theme === "dark" ? "light" : "dark")}>{theme === "dark" ? <Sun /> : <Moon />}{theme === "dark" ? "Light theme" : "Dark theme"}</DropdownMenuItem>
      <DropdownMenuSeparator />
      <DropdownMenuItem onSelect={() => {
        useAuthStore.getState().logout();
        useSceneStore.getState().reset();
        useFolderStore.getState().reset();
        useSearchStore.getState().reset();
        useWorkspaceStore.setState({ workspaces: [], activeWorkspaceId: null, isLoading: false });
        navigate("/");
      }}><LogOut />Sign out</DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>;
}
