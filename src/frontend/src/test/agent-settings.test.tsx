import {render,screen,fireEvent,waitFor,cleanup} from '@testing-library/react';
import {afterEach,beforeEach,describe,expect,it,vi} from 'vitest';
import {AgentSettings} from '../components/collaboration/agent-settings';
import api from '../lib/api';
vi.mock('../lib/api',()=>({default:{get:vi.fn(),post:vi.fn(),delete:vi.fn()}}));
afterEach(cleanup);beforeEach(()=>{vi.resetAllMocks();vi.mocked(api.get).mockResolvedValue({data:[]});vi.mocked(api.post).mockResolvedValue({data:{id:'key-1',scopes:['scene:read'],sceneIds:[],createdAt:'2026-10-16',revokedAt:null,key:'private-once'}})});
describe('agent settings',()=>{
 it('keeps management owner-only',()=>{render(<AgentSettings workspaceId="w" isOwner={false}/>);expect(screen.getByText(/Only workspace owners/)).toBeInTheDocument();expect(api.get).not.toHaveBeenCalled()});
 it('shows a new key once and clears it when dismissed or workspace changes',async()=>{const view=render(<AgentSettings workspaceId="w" isOwner/>);fireEvent.click(screen.getByRole('button',{name:'Create key'}));await screen.findByDisplayValue('private-once');fireEvent.click(screen.getByRole('button',{name:"I've saved this key"}));expect(screen.queryByDisplayValue('private-once')).not.toBeInTheDocument();await waitFor(()=>expect(api.post).toHaveBeenCalled());fireEvent.click(screen.getByRole('button',{name:'Create key'}));await screen.findByDisplayValue('private-once');view.rerender(<AgentSettings workspaceId="other" isOwner/>);expect(screen.queryByDisplayValue('private-once')).not.toBeInTheDocument()});
});
