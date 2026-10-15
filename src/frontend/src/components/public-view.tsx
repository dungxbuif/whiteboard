import { LockKeyhole } from "lucide-react";
import { useEffect, useState } from "react";
import type { ComponentType } from "react";

import "@excalidraw/excalidraw/index.css";

import api from "@/lib/api";
import type { SceneContent } from "@/store/scene-store";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";

type PublicScene = { title: string; content: SceneContent };
type ExcalidrawModule = typeof import("@excalidraw/excalidraw");

export function PublicView({ token }: { token: string }) {
  const [scene, setScene] = useState<PublicScene | null>(null);
  const [module, setModule] = useState<ExcalidrawModule | null>(null);
  const [password, setPassword] = useState("");
  const [requiresPassword, setRequiresPassword] = useState(false);
  const [error, setError] = useState("");

  async function loadScene(candidatePassword = "") {
    setError("");
    try {
      const response = await api.get<PublicScene>(`/public/scenes/${token}`, candidatePassword ? { params: { password: candidatePassword } } : undefined);
      setScene(response.data);
      setRequiresPassword(false);
    } catch (requestError: any) {
      if (requestError.response?.status === 401) setRequiresPassword(true);
      else setError("This shared scene is unavailable.");
    }
  }

  useEffect(() => {
    void import("@excalidraw/excalidraw").then(setModule);
    void loadScene();
  }, [token]);

  if (error) return <main className="grid min-h-screen place-items-center p-6"><p className="text-destructive">{error}</p></main>;
  if (requiresPassword && !scene) {
    return (
      <main className="grid min-h-screen place-items-center bg-muted/30 p-6">
        <Card className="w-full max-w-sm">
          <CardHeader><CardTitle className="flex items-center gap-2"><LockKeyhole size={18} /> Password required</CardTitle></CardHeader>
          <CardContent className="grid gap-3">
            <Input autoFocus onChange={(event) => setPassword(event.target.value)} placeholder="Share password" type="password" value={password} />
            <Button onClick={() => void loadScene(password)} type="button">Open shared scene</Button>
          </CardContent>
        </Card>
      </main>
    );
  }
  if (!scene || !module) return <main className="grid min-h-screen place-items-center">Loading shared scene...</main>;

  const Excalidraw = module.Excalidraw as ComponentType<Record<string, unknown>>;
  return (
    <main className="grid h-screen grid-rows-[56px_minmax(0,1fr)] bg-background">
      <header className="flex items-center border-b bg-card px-4"><h1 className="font-semibold">{scene.title}</h1><span className="ml-auto text-sm text-muted-foreground">Read-only</span></header>
      <section aria-label="Public Excalidraw canvas" className="min-h-0"><Excalidraw initialData={scene.content} name={scene.title} viewModeEnabled /></section>
    </main>
  );
}
