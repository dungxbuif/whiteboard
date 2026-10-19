import {expect,it} from 'vitest';
import {Presence} from './presence';
import {PresenceTransportFixture,participant} from './presence-fixture';
it('keeps same-user tabs distinct, removes departed cursors and clears connection state',()=>{
 const transport=new PresenceTransportFixture(),presence=new Presence(transport,()=> 'Alice');presence.start();
 transport.emit('roster',{participants:[participant('a'),participant('b')]});transport.emit('awareness',{connectionId:'b',principalId:'same-user',cursor:{x:10,y:20},selectedIds:['shape']});
 expect(presence.getSnapshot().participants).toHaveLength(2);expect(presence.getSnapshot().cursors.b.cursor).toEqual({x:10,y:20});
 transport.emit('roster',{participants:[participant('a')]});expect(presence.getSnapshot().cursors.b).toBeUndefined();
 transport.status('offline');expect(presence.getSnapshot().participants).toEqual([]);expect(presence.getSnapshot().connectionId).toBeUndefined();presence.stop();
});
it('refresh on the same connection preserves presence but reconnect clears it',()=>{
 const transport=new PresenceTransportFixture(),presence=new Presence(transport,()=> 'Alice');presence.start();transport.emit('roster',{participants:[participant('a'),participant('b')]});
 transport.status('connecting');expect(presence.getSnapshot().participants).toHaveLength(2);
 transport.status('connecting',undefined);expect(presence.getSnapshot().participants).toHaveLength(0);presence.stop();
});
