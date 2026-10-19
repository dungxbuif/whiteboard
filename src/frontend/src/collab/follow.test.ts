import {expect,it,vi} from 'vitest';
import {Presence} from './presence';
import {PresenceTransportFixture,participant} from './presence-fixture';
it('rejects self and cycle follows, ignores stale viewport, and clears when target exits',()=>{
 const t=new PresenceTransportFixture(),p=new Presence(t,()=> 'Alice');p.start();t.emit('roster',{participants:[participant('a'),participant('b','a'),participant('c')]});
 expect(p.follow('a')).toBe(false);expect(p.follow('b')).toBe(false);expect(p.follow('c')).toBe(true);
 const update=vi.fn();p.subscribeViewport(update);t.emit('follow',{connectionId:'a',targetConnectionId:'c'});
 t.emit('viewport',{connectionId:'b',scrollX:1,scrollY:2,zoom:1});expect(update).not.toHaveBeenCalled();
 t.emit('viewport',{connectionId:'c',scrollX:3,scrollY:4,zoom:2});expect(update).toHaveBeenCalledWith({scrollX:3,scrollY:4,zoom:2});
 t.emit('roster',{participants:[participant('a')]});expect(p.getSnapshot().following).toBeNull();p.stop();
});
