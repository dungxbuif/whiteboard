import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';

export function WaitingRoom({status='waiting',error,onLeave,onRejoin}:{status?:'joining'|'waiting'|'denied'|'removed'|'ended';error?:string;onLeave:()=>void;onRejoin?:()=>void}){
 const title=status==='joining'?'Joining presentation…':status==='waiting'?'Waiting for the host':status==='denied'?'The host did not admit you':status==='removed'?'You were removed':'Presentation ended';
 return <main className="grid min-h-[100dvh] place-items-center bg-muted/30 p-4"><Card className="w-full max-w-sm"><CardHeader><CardTitle>{title}</CardTitle></CardHeader><CardContent className="grid gap-4"><p className="text-sm text-muted-foreground">{status==='waiting'?'No scene, participant, voice, screen, or asset data is loaded until the host admits this browser.':error??'Ask the host for a current invite if you want to join again.'}</p>{error?<p className="text-sm text-destructive" role="alert">{error}</p>:null}{onRejoin?<Button className="min-h-11" onClick={onRejoin}>Rejoin waiting room</Button>:null}<Button className="min-h-11" onClick={onLeave} variant="outline">Leave</Button></CardContent></Card></main>;
}
