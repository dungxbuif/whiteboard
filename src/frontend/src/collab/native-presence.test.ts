import {expect,it,vi} from 'vitest';
import {Presence} from './presence';
import {PresenceTransportFixture,participant} from './presence-fixture';
import {bindNativePresence} from './native-presence';
it('projects distinct tabs without echoing server follow state and follows ephemeral camera state only',async()=>{
 const t=new PresenceTransportFixture(),presence=new Presence(t,()=> 'Alice');presence.start();
 let nativeTarget:any=null;
 const callbacks:any={},api:any={updateScene:vi.fn((value:any)=>{if(!('userToFollow' in (value.appState??{})))return;const next=value.appState.userToFollow;if(nativeTarget===next)return;if(nativeTarget)callbacks.follow?.({action:'UNFOLLOW',userToFollow:nativeTarget});nativeTarget=next;if(nativeTarget)callbacks.follow?.({action:'FOLLOW',userToFollow:nativeTarget});}),getAppState:()=>({scrollX:1,scrollY:2,zoom:{value:1},userToFollow:nativeTarget}),onScrollChange:(f:any)=>{callbacks.scroll=f;return vi.fn();},onUserFollow:(f:any)=>{callbacks.follow=f;return vi.fn();},onChange:(f:any)=>{callbacks.change=f;return vi.fn();}};
 const stop=bindNativePresence(presence,api,'NEVER' as never);
 t.emit('roster',{participants:[participant('a','b'),participant('b')]});
 await Promise.resolve();
 const projection=api.updateScene.mock.calls.at(-1)[0];expect(projection.collaborators.size).toBe(2);expect(projection.appState.userToFollow.socketId).toBe('b');expect(projection.collaborators.get('a').isCurrentUser).toBe(true);
 expect(t.sent.filter(value=>value.type==='follow')).toHaveLength(0);
 t.emit('roster',{participants:[participant('a','b'),participant('b')]});await Promise.resolve();expect(t.sent.filter(value=>value.type==='follow')).toHaveLength(0);
 t.emit('viewport',{connectionId:'b',scrollX:50,scrollY:60,zoom:2});expect(api.updateScene).toHaveBeenLastCalledWith({appState:{scrollX:50,scrollY:60,zoom:{value:2}},captureUpdate:'NEVER'});
 callbacks.follow({action:'UNFOLLOW',userToFollow:{socketId:'b'}});await Promise.resolve();expect(t.sent.at(-1)?.payload).toEqual({targetConnectionId:null});
 t.emit('roster',{participants:[participant('a')]});expect(api.updateScene.mock.calls.at(-1)[0].appState.userToFollow).toBeNull();
 stop();expect(api.updateScene.mock.calls.at(-1)[0].collaborators.size).toBe(0);presence.stop();
});
