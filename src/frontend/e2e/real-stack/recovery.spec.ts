import { test, expect } from './fixtures';
import { openScene, draw, elements, pending } from './browser';
import { writeFileSync, readFileSync, existsSync } from 'node:fs';
import { join } from 'node:path';
import { until } from './harness';
test('native concurrent/offline edits replay after reload; offline downgrade retains exportable queue',async({browser,stack},info)=>{
  const scene=await stack.scene();const ca=await browser.newContext(),cb=await browser.newContext();
  try {
    const a=await ca.newPage(),b=await cb.newPage();await openScene(a,stack,scene);await openScene(b,stack,scene,stack.editor);
    await Promise.all([draw(a,250),draw(b,450)]);await expect.poll(async()=>(await elements(a)).length).toBe(2);await expect.poll(async()=>JSON.stringify(await elements(b))).toBe(JSON.stringify(await elements(a)));
    await cb.setOffline(true);await b.evaluate(()=>window.dispatchEvent(new Event('offline')));await expect(b.getByRole('status')).toContainText('Offline');await draw(b,650);await expect.poll(async()=>(await pending(b)).length).toBeGreaterThan(0);
    await cb.setOffline(false);await b.reload();await expect(b.getByRole('status')).toHaveText('Saved');await expect.poll(async()=>(await elements(a)).length).toBe(3);
    await cb.setOffline(true);await b.evaluate(()=>window.dispatchEvent(new Event('offline')));await draw(b,800);await expect.poll(async()=>(await pending(b)).length).toBeGreaterThan(0);
    await stack.json('/workspaces/'+stack.owner.workspaceId+'/members/'+stack.editor.id,stack.owner.token,stack.owner.workspaceId,'PATCH',{role:'viewer'});
    await cb.setOffline(false);await b.evaluate(()=>window.dispatchEvent(new Event('online')));await expect(b.getByRole('status')).toHaveText('Read-only',{timeout:20000});expect((await pending(b)).length).toBeGreaterThan(0);
    await expect.poll(async()=>(await elements(a)).length).toBe(3);
    const download=b.waitForEvent('download');await b.getByRole('button',{name:'Export recovery',exact:true}).click();const file=await download;await file.saveAs(info.outputPath('offline-downgrade-recovery.json'));const recovery=JSON.parse(readFileSync(info.outputPath('offline-downgrade-recovery.json'),'utf8'));expect(recovery.scopes.some((s:any)=>s.pending.length>0)).toBe(true);
  } finally {await Promise.all([ca.close(),cb.close()]);await stack.json('/workspaces/'+stack.owner.workspaceId+'/members/'+stack.editor.id,stack.owner.token,stack.owner.workspaceId,'PATCH',{role:'editor'});}
});
test('actual child dies after durable commit before ACK; browser replays same receipt once after restart',async({page,stack})=>{
  const scene=await stack.scene();await openScene(page,stack,scene);writeFileSync(join(stack.dir,'kill-before-ack'),'armed');
  await draw(page);await until(()=>existsSync(join(stack.dir,'committed-before-ack.json')),20000,'test hook after DB commit');
  const ack=JSON.parse(readFileSync(join(stack.dir,'committed-before-ack.json'),'utf8'));
  const receipt=await stack.pool.query('SELECT sequence::text FROM collab_receipts WHERE scene_id=$1 AND request_id=$2',[scene.id,ack.requestId]);expect(receipt.rows).toHaveLength(1);expect(receipt.rows[0].sequence).toBe(ack.payload.sequence);
  await expect.poll(async()=>(await pending(page)).length).toBeGreaterThan(0);
  await until(()=>stack.runtime!.exitCode!==null||stack.runtime!.signalCode!==null);await stack.startRuntime();await page.reload();await expect(page.getByRole('status')).toHaveText('Saved',{timeout:20000});expect(await pending(page)).toHaveLength(0);expect(await elements(page)).toHaveLength(1);
  expect((await stack.pool.query('SELECT count(*) FROM collab_mutation_audit WHERE scene_id=$1 AND request_id=$2',[scene.id,ack.requestId])).rows[0].count).toBe('1');
  expect((await stack.pool.query('SELECT sequence::text FROM collab_documents WHERE scene_id=$1',[scene.id])).rows[0].sequence).toBe(ack.payload.sequence);
});
test('new durable document epoch refuses stale browser pending replay',async({page,context,stack})=>{
  const scene=await stack.scene();await openScene(page,stack,scene);await draw(page);await expect(page.getByRole('status')).toHaveText('Saved');
  await context.setOffline(true);await page.evaluate(()=>window.dispatchEvent(new Event('offline')));await draw(page,500);await expect.poll(async()=>(await pending(page)).length).toBeGreaterThan(0);
  await stack.stop(stack.runtime);await stack.pool.query('UPDATE collab_documents SET document_id=gen_random_uuid() WHERE scene_id=$1',[scene.id]);await stack.startRuntime();
  await context.setOffline(false);await page.reload();await expect(page.getByRole('status')).toHaveText('Recovery required');expect((await pending(page)).length).toBeGreaterThan(0);await expect(page.getByRole('button',{name:'Export recovery',exact:true})).toBeVisible();
});

test('account switch leaves prior principal pending namespace intact and never replays it',async({page,context,stack})=>{
  const scene=await stack.scene();await openScene(page,stack,scene);await context.setOffline(true);await page.evaluate(()=>window.dispatchEvent(new Event('offline')));await draw(page);await expect.poll(async()=>(await pending(page)).length).toBeGreaterThan(0);
  const original=await pending(page);await page.evaluate(async()=>{(await import('/src/store/auth-store.ts' as string)).default.getState().logout();});await context.setOffline(false);
  await page.goto(stack.baseURL+'/');await page.getByLabel('Email',{exact:true}).fill(stack.outsider.email);await page.getByLabel('Password',{exact:true}).fill(stack.outsider.password);await page.getByRole('button',{name:'Sign in',exact:true}).click();await expect.poll(()=>page.evaluate(()=>!!localStorage.getItem('auth_token'))).toBe(true);
  const retained=await page.evaluate(async scope=>{const {offlineStore}=await import('/src/collab/offline-store.ts' as string);return (await offlineStore.pending(scope)).map((p:any)=>p.requestId);},original[0].scope);expect(retained).toEqual(original.map((p:any)=>p.requestId));expect((await stack.pool.query('SELECT count(*) FROM collab_receipts WHERE scene_id=$1',[scene.id])).rows[0].count).toBe('0');
});
