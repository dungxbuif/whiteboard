import { test, expect } from './fixtures';
test('real password, invitation and effective role authorization', async ({ stack }) => {
  const scene = await stack.scene();
  for (const [account, access, owner] of [[stack.owner, 'edit', true], [stack.editor, 'edit', false], [stack.viewer, 'view', false]] as const) {
    const result = await stack.json('/scenes/' + scene.id, account.token, stack.owner.workspaceId);
    expect(result.access).toBe(access); expect(result.owner).toBe(owner);
  }
  expect((await stack.request('/workspaces/' + stack.owner.workspaceId + '/invitations', stack.editor.token, stack.owner.workspaceId, 'POST', { email: 'denied@example.com', role: 'editor' })).status).toBe(403);
  expect((await stack.request('/scenes/' + scene.id, stack.viewer.token, stack.owner.workspaceId, 'PATCH', { content: { elements: [], appState: {}, files: {} } })).status).toBe(403);
  expect((await stack.request('/scenes/' + scene.id, stack.outsider.token, stack.owner.workspaceId)).status).toBe(404);
});

import { login, openScene, draw, elements } from './browser';
test('owner/editor/viewer rendered permissions, native edits, UUID and short alias reload', async ({ browser, stack }, info) => {
  const scene=await stack.scene();
  const contexts=await Promise.all([browser.newContext(),browser.newContext(),browser.newContext()]);
  try {
    const [owner,editor,viewer]=await Promise.all(contexts.map(c=>c.newPage()));
    await openScene(owner,stack,scene); await openScene(editor,stack,scene,stack.editor);await openScene(viewer,stack,scene,stack.viewer);
    await expect(viewer.getByTitle('Rectangle — R or 2',{exact:true})).toHaveCount(0);
    await draw(editor);await expect(editor.getByRole('status')).toHaveText('Saved');
    await expect.poll(async()=>(await elements(owner)).length).toBe(1); await expect.poll(async()=>(await elements(viewer)).length).toBe(1);
    expect(await elements(viewer)).toEqual(await elements(editor));
    const spaces=await stack.json('/workspaces',stack.owner.token); const space=spaces.find((w:any)=>w.id===stack.owner.workspaceId);
    expect(space.publicId).toHaveLength(21); expect(scene.publicId).toHaveLength(21);
    await owner.goto(stack.baseURL+'/w/'+space.publicId+'/scenes/'+scene.publicId); await expect(owner.getByRole('status')).toHaveText('Saved'); await owner.reload(); await expect(owner.getByRole('status')).toHaveText('Saved');expect((await elements(owner)).length).toBe(1);
    await owner.goto(stack.baseURL+'/w/'+space.publicId+'/settings/members');await expect(owner.getByRole('button',{name:'Invite member'})).toBeVisible();
    await viewer.goto(stack.baseURL+'/w/'+space.publicId);await expect(viewer.getByRole('button',{name:'New scene'})).toHaveCount(0);await expect(viewer.getByRole('link',{name:'Members',exact:true})).toHaveCount(0);
    await editor.goto(stack.baseURL+'/w/'+space.publicId+'/settings/members');await expect(editor.getByRole('alert')).toContainText('Only workspace owners');
    await viewer.screenshot({path:info.outputPath('viewer-dashboard.png')});
  } finally {await Promise.all(contexts.map(c=>c.close()));}
});
test('public view/edit sessions, bad password preserves login, guest identity survives reload',async({browser,stack})=>{
  const scene=await stack.scene();const view=await stack.json('/scenes/'+scene.id+'/share-links',stack.owner.token,stack.owner.workspaceId,'POST',{access:'view'});
  const edit=await stack.json('/scenes/'+scene.id+'/share-links',stack.owner.token,stack.owner.workspaceId,'POST',{access:'edit',password:'e1-secret-pass'});
  const context=await browser.newContext(); const page=await context.newPage();
  try {
    await login(page,stack);const original=await page.evaluate(()=>localStorage.getItem('auth_token'));
    await page.goto(stack.baseURL+'/p/'+edit.token);await page.getByLabel('Share password').fill('incorrect');await page.getByRole('button',{name:'Open shared scene'}).click();await expect(page.getByRole('alert')).toBeVisible();expect(await page.evaluate(()=>localStorage.getItem('auth_token'))).toBe(original);
    await page.getByLabel('Share password').fill('e1-secret-pass');await page.getByRole('button',{name:'Open shared scene'}).click();await page.getByLabel('Display name').fill('E1 Guest');await page.getByRole('button',{name:'Open shared scene'}).click();await expect(page.getByRole('status')).toHaveText('Saved');
    const saved=()=>page.evaluate(token=>JSON.parse(sessionStorage.getItem('devtool:guest-session:'+token)!).session,edit.token);
    const before=await saved();await draw(page);await expect(page.getByRole('status')).toHaveText('Saved');await page.reload();await expect(page.getByRole('status')).toHaveText('Saved');expect((await saved()).principalId).toBe(before.principalId);
    expect(await page.evaluate(()=>localStorage.getItem('auth_token'))).toBe(original);
    const viewPage=await context.newPage();await viewPage.goto(stack.baseURL+'/p/'+view.token);await expect(viewPage.getByRole('status')).toHaveText('Read-only');await expect(viewPage.getByTitle('Rectangle — R or 2',{exact:true})).toHaveCount(0);
    const guest=await stack.json('/public/scene-session','','','POST',{token:view.token});
    expect((await stack.request('/scenes/'+scene.id+'/assets/upload-capability',guest.token,stack.owner.workspaceId,'POST',{assets:[{fileId:'denied',mimeType:'image/png',byteSize:4}]})).status).toBe(403);
    await context.setOffline(true);await page.evaluate(()=>window.dispatchEvent(new Event('offline')));await expect(page.getByRole('status')).toContainText('Offline');await draw(page,500);
    const oldPending=()=>page.evaluate(async({principal,workspace,scene})=>{const {offlineStore}=await import('/src/collab/offline-store.ts' as string);const scopes=await offlineStore.scopes(principal,workspace,scene);return (await Promise.all(scopes.map((s:any)=>offlineStore.pending(s)))).flat().length;},{principal:before.principalId,workspace:stack.owner.workspaceId,scene:scene.id});
    await expect.poll(oldPending).toBeGreaterThan(0);const durable=(await stack.pool.query('SELECT sequence::text FROM collab_documents WHERE scene_id=$1',[scene.id])).rows[0].sequence;
    await stack.json('/scenes/'+scene.id+'/share-links/'+edit.id,stack.owner.token,stack.owner.workspaceId,'DELETE');await context.setOffline(false);await page.evaluate(()=>window.dispatchEvent(new Event('online')));await expect(page.getByRole('status')).toContainText('Access revoked',{timeout:20000});expect(await oldPending()).toBeGreaterThan(0);
    const fresh=await stack.json('/scenes/'+scene.id+'/share-links',stack.owner.token,stack.owner.workspaceId,'POST',{access:'edit'});await page.goto(stack.baseURL+'/p/'+fresh.token);await page.getByLabel('Display name').fill('New guest identity');await page.getByRole('button',{name:'Open shared scene'}).click();await expect(page.getByRole('status')).toHaveText('Saved');expect(await oldPending()).toBeGreaterThan(0);expect((await stack.pool.query('SELECT sequence::text FROM collab_documents WHERE scene_id=$1',[scene.id])).rows[0].sequence).toBe(durable);

    expect((await stack.request('/scenes/'+scene.id+'/room-token',before.token,stack.owner.workspaceId,'POST',{})).status).toBe(404);
  } finally {await context.close();}
});

test('native frame creation and presentation order survive durable reload',async({page,stack})=>{
  const scene=await stack.scene();await openScene(page,stack,scene);await draw(page,350);await expect(page.getByRole('status')).toHaveText('Saved');
  await page.keyboard.press('f');const box=await page.locator('canvas').first().boundingBox();await page.mouse.move(box!.x+300,box!.y+100);await page.mouse.down();await page.mouse.move(box!.x+600,box!.y+350,{steps:5});await page.mouse.up();
  await expect.poll(async()=>(await elements(page)).filter((e:any)=>e.type==='frame').length).toBe(1);await expect(page.getByRole('status')).toHaveText('Saved');
  const before=await elements(page);await page.reload();await expect(page.getByRole('status')).toHaveText('Saved');expect(await elements(page)).toEqual(before);await page.getByRole('button',{name:'Present',exact:true}).click();await expect(page.getByRole('button',{name:'Exit presentation'}).first()).toBeVisible();
});
