import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { RestoreDialog } from './restore-dialog';

const version={id:'version-1',sceneId:'scene-1',documentId:'document-1',sequence:'7',hash:'a'.repeat(64),pinned:true,kind:'manual',creator:{kind:'user',id:'user-1'},metadataRevision:'1',createdAt:'2026-10-20T00:00:00Z',updatedAt:'2026-10-20T00:00:00Z'};
const ref={workspaceId:'workspace-1',sceneId:'scene-1'};

describe('RestoreDialog retry identity',()=>{
 it('reuses one request ID after an uncertain response',async()=>{
  const restore=vi.fn().mockRejectedValueOnce({response:{status:503}}).mockResolvedValue({requestId:'r',sequence:'8',stateVector:'AA==',preRestoreVersionId:'p'});
  render(<RestoreDialog api={{restore}} currentSequence="7" onCancel={vi.fn()} onRestored={vi.fn()} sceneRef={ref} version={version}/>);
  fireEvent.click(screen.getByRole('button',{name:'Confirm restore'}));expect(await screen.findByRole('alert')).toHaveTextContent(/acknowledgement is uncertain/i);
  fireEvent.click(screen.getByRole('button',{name:'Confirm restore'}));await waitFor(()=>expect(restore).toHaveBeenCalledTimes(2));
  expect(restore.mock.calls[1][2].requestId).toBe(restore.mock.calls[0][2].requestId);
 });
 it('waits for a fresh sequence after a definitive conflict and then rebases request identity',async()=>{
  const restore=vi.fn().mockRejectedValueOnce({response:{status:409,data:{code:'CONFLICT'}}}).mockResolvedValue({requestId:'r',sequence:'9',stateVector:'AA==',preRestoreVersionId:'p'});
  const props={api:{restore},onCancel:vi.fn(),onRestored:vi.fn(),sceneRef:ref,version};const view=render(<RestoreDialog {...props} currentSequence="7"/>);
  fireEvent.click(screen.getByRole('button',{name:'Confirm restore'}));expect(await screen.findByRole('alert')).toHaveTextContent(/waiting for the latest collaboration sequence/i);expect(screen.getByRole('button',{name:'Confirm restore'})).toBeDisabled();
  const first=restore.mock.calls[0][2].requestId;view.rerender(<RestoreDialog {...props} currentSequence="8"/>);expect(screen.getByRole('button',{name:'Confirm restore'})).toBeEnabled();
  fireEvent.click(screen.getByRole('button',{name:'Confirm restore'}));await waitFor(()=>expect(restore).toHaveBeenCalledTimes(2));expect(restore.mock.calls[1][2].requestId).not.toBe(first);
 });
});
