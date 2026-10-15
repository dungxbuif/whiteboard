import { Mic, MicOff, Radio, Users } from "lucide-react";

import { Button } from "@/components/ui/button";
import useRealtimeStore from "@/store/realtime-store";

export function CollaborationToolbar() {
  const status = useRealtimeStore((state) => state.status);
  const peerCount = Object.keys(useRealtimeStore((state) => state.peers)).length;
  const voiceMuted = useRealtimeStore((state) => state.voiceMuted);
  const toggleVoiceMute = useRealtimeStore((state) => state.toggleVoiceMute);
  const isConnected = status === "connected";
  return (
    <div className="flex items-center gap-2 rounded-md border bg-card/95 px-2 py-1 shadow-sm" aria-label="Collaboration status">
      <span className={`flex items-center gap-1 text-xs ${isConnected ? "text-emerald-600" : "text-muted-foreground"}`}>
        <Radio size={13} aria-hidden="true" /> {isConnected ? "Live" : "Offline"}
      </span>
      <span className="flex items-center gap-1 text-xs text-muted-foreground" title={`${peerCount} collaborators`}>
        <Users size={13} aria-hidden="true" /> {peerCount}
      </span>
      <Button aria-label={voiceMuted ? "Unmute voice" : "Mute voice"} onClick={toggleVoiceMute} size="icon" type="button" variant="ghost">
        {voiceMuted ? <MicOff size={15} aria-hidden="true" /> : <Mic size={15} aria-hidden="true" />}
      </Button>
    </div>
  );
}
