import type { Scene } from '@/store/scene-store';
import useWorkspaceStore from '@/store/workspace-store';
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from '@/components/ui/dialog';
import { ShareLinks } from '@/components/collaboration/share-links';
export function ShareDialog({ scene, open, onOpenChange }: { scene: Scene; open: boolean; onOpenChange: (open: boolean) => void }) {
  const workspaceId = useWorkspaceStore(state => state.activeWorkspaceId);
  return <Dialog open={open} onOpenChange={onOpenChange}><DialogContent className="max-h-[90dvh] overflow-y-auto"><DialogHeader><DialogTitle>Share {scene.title}</DialogTitle><DialogDescription>Create independent view or edit links. Each link can expire or be revoked separately.</DialogDescription></DialogHeader>{open && workspaceId && scene.owner && <ShareLinks key={workspaceId + scene.id} workspaceId={workspaceId} sceneId={scene.id} />}</DialogContent></Dialog>;
}
