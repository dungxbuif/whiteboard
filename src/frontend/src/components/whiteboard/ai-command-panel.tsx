import { Bot, Code2, ImagePlus, Loader2, MessageCircle, WandSparkles } from "lucide-react";
import { useRef, useState } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import type { SceneContent } from "@/store/scene-store";
import useAIStore from "@/store/ai-store";

interface AICommandPanelProps {
  scene: SceneContent;
  onApplyDiagram: (scene: SceneContent) => void;
}

export function AICommandPanel({ scene, onApplyDiagram }: AICommandPanelProps) {
  const [prompt, setPrompt] = useState("");
  const [imageDataUrl, setImageDataUrl] = useState<string | undefined>();
  const fileInput = useRef<HTMLInputElement>(null);
  const isLoading = useAIStore((state) => state.isLoading);
  const error = useAIStore((state) => state.error);
  const result = useAIStore((state) => state.result);
  const runChat = useAIStore((state) => state.runChat);
  const generateDiagram = useAIStore((state) => state.generateDiagram);
  const toCode = useAIStore((state) => state.toCode);

  const readImage = (file?: File) => {
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => setImageDataUrl(typeof reader.result === "string" ? reader.result : undefined);
    reader.readAsDataURL(file);
  };

  return (
    <aside className="absolute bottom-4 left-4 z-10 w-[calc(100vw-2rem)] max-w-[420px] rounded-lg border bg-card/95 p-3 shadow-lg backdrop-blur" aria-label="AI command panel">
      <div className="mb-2 flex items-center gap-2 text-sm font-semibold"><Bot size={16} aria-hidden="true" /> AI assistant</div>
      <div className="flex gap-2">
        <Input aria-label="AI prompt" onChange={(event) => setPrompt(event.target.value)} placeholder="Describe a diagram or ask about this board" value={prompt} />
        <Button aria-label="Attach image" onClick={() => fileInput.current?.click()} size="icon" type="button" variant="outline"><ImagePlus size={16} aria-hidden="true" /></Button>
        <input ref={fileInput} accept="image/*" className="hidden" onChange={(event) => readImage(event.target.files?.[0])} type="file" />
      </div>
      <div className="mt-2 flex flex-wrap gap-2">
        <Button disabled={isLoading || !prompt.trim()} onClick={() => void runChat(prompt, imageDataUrl)} size="sm" type="button" variant="outline"><MessageCircle size={14} aria-hidden="true" /> Ask</Button>
        <Button disabled={isLoading || !prompt.trim()} onClick={async () => { const generated = await generateDiagram(prompt, imageDataUrl); if (generated) onApplyDiagram(generated); }} size="sm" type="button"><WandSparkles size={14} aria-hidden="true" /> Diagram</Button>
        <Button disabled={isLoading || !prompt.trim()} onClick={() => void toCode(prompt, scene)} size="sm" type="button" variant="secondary"><Code2 size={14} aria-hidden="true" /> Code</Button>
        {isLoading ? <Loader2 className="animate-spin self-center text-muted-foreground" size={16} aria-label="Loading" /> : null}
      </div>
      {error ? <p className="mt-2 text-xs text-destructive" role="alert">{error}</p> : null}
      {result ? <pre className="mt-3 max-h-44 overflow-auto rounded bg-muted p-2 text-xs whitespace-pre-wrap" data-testid="ai-result">{typeof result.content === "string" ? result.content : JSON.stringify(result.content, null, 2)}</pre> : null}
    </aside>
  );
}
