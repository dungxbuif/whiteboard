import {fireEvent,render,screen,waitFor} from '@testing-library/react';
import {describe,expect,it,vi} from 'vitest';
import type {SCIMClient} from '@/lib/scim-api';
import {SCIMSettings} from './scim-settings';

function client(configured=false):SCIMClient{return{status:vi.fn().mockResolvedValue(configured?{configured:true,prefix:'scim_abcdef',createdAt:'2026-10-20T00:00:00Z'}:{configured:false}),rotate:vi.fn().mockResolvedValue({token:'scim_secret-once',prefix:'scim_secret',createdAt:'2026-10-20T00:00:00Z'}),revoke:vi.fn().mockResolvedValue(undefined)}}
describe('SCIMSettings',()=>{
 it('shows the token once and copies it',async()=>{const api=client();const writeText=vi.fn().mockResolvedValue(undefined);Object.defineProperty(navigator,'clipboard',{configurable:true,value:{writeText}});render(<SCIMSettings api={api} workspaceId="workspace-1"/>);expect(await screen.findByText(/no scim token/i)).toBeInTheDocument();fireEvent.click(screen.getByRole('button',{name:'Create token'}));expect(await screen.findByDisplayValue('scim_secret-once')).toBeInTheDocument();fireEvent.click(screen.getByRole('button',{name:'Copy token'}));await waitFor(()=>expect(writeText).toHaveBeenCalledWith('scim_secret-once'));expect(screen.getByText(/shown only once/i)).toBeInTheDocument()});
 it('revokes an active token and explains access boundaries',async()=>{const api=client(true);render(<SCIMSettings api={api} workspaceId="workspace-1"/>);fireEvent.click(await screen.findByRole('button',{name:'Revoke token'}));await waitFor(()=>expect(api.revoke).toHaveBeenCalledWith('workspace-1'));expect(await screen.findByRole('status')).toHaveTextContent(/revoked/i);expect(screen.getByText(/manual access and other workspaces are preserved/i)).toBeInTheDocument()});
});
