import {fireEvent,render,screen,waitFor} from '@testing-library/react';
import {describe,expect,it,vi} from 'vitest';
import {LibraryPanel} from './library-panel';

it('searches authorized workspace items and inserts a selected immutable revision',async()=>{
 const api={list:vi.fn().mockResolvedValue({items:[{id:'one',name:'Flow',currentRevision:'2',createdAt:'',updatedAt:''}]}),insert:vi.fn().mockResolvedValue({requestId:'r',sequence:'8',stateVector:'v',elementIds:['e']}),import:vi.fn(),publish:vi.fn(),revision:vi.fn(),export:vi.fn(),cancelPending:vi.fn()};
 const inserted=vi.fn();render(<LibraryPanel api={api} canManage currentSequence="7" onInserted={inserted} sceneId="scene" workspaceId="workspace"/>);expect(await screen.findByText('Flow')).toBeInTheDocument();fireEvent.change(screen.getByLabelText('Search workspace libraries'),{target:{value:'flo'}});fireEvent.click(screen.getByRole('button',{name:'Insert Flow'}));await waitFor(()=>expect(api.insert).toHaveBeenCalledWith('workspace','scene','one',2,'7'));expect(inserted).toHaveBeenCalled();
});
