import {cleanup,fireEvent,render,screen,waitFor} from '@testing-library/react';
import {afterEach,beforeEach,describe,expect,it,vi} from 'vitest';
import {ShareLinks} from '../components/collaboration/share-links';
import api from '../lib/api';

vi.mock('../lib/api',()=>({default:{get:vi.fn(),post:vi.fn(),delete:vi.fn()}}));

afterEach(cleanup);
beforeEach(()=>{
 vi.resetAllMocks();
 vi.mocked(api.get).mockResolvedValue({data:[{id:'existing',access:'view',allowComments:true,active:true,expiresAt:null}]});
 vi.mocked(api.post).mockResolvedValue({data:{id:'created',access:'view',allowComments:true,active:true,expiresAt:null,token:'secret'}});
});

describe('share links',()=>{
 it('creates an independently comment-enabled view link and displays stored policy',async()=>{
  render(<ShareLinks workspaceId="workspace" sceneId="scene"/>);
  expect(await screen.findByText(/Comments allowed/)).toBeInTheDocument();
  fireEvent.click(screen.getByRole('checkbox',{name:'Allow comments on view link'}));
  fireEvent.click(screen.getByRole('button',{name:'Create view link'}));
  await waitFor(()=>expect(api.post).toHaveBeenCalledWith('/scenes/scene/share-links',{access:'view',allowComments:true},{headers:{'X-Workspace-ID':'workspace'}}));
 });
});
