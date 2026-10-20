import {describe,expect,it,vi} from 'vitest';
import {createLibraryClient,libraryPath} from './library-api';

describe('workspace library API',()=>{
 it('keeps one request id while an insertion outcome is uncertain',async()=>{
  const post=vi.fn().mockRejectedValueOnce(new Error('offline')).mockResolvedValueOnce({data:{requestId:'ignored',sequence:'8',stateVector:'v',elementIds:['e']}});const client=createLibraryClient({get:vi.fn(),post,put:vi.fn()});
  await expect(client.insert('workspace','scene','item',1,'7')).rejects.toThrow('offline');
  await client.insert('workspace','scene','item',1,'7');
  expect(post.mock.calls[0][1].requestId).toBe(post.mock.calls[1][1].requestId);
 });
 it('uses the canonical workspace route',()=>{expect(libraryPath('team')).toBe('/w/team/libraries')});
});
