import { describe, expect, it, vi } from 'vitest';
import { bindFeatureInvalidation, FeatureInvalidationCoordinator } from './feature-invalidation';

function deferred() {
  let resolve!: () => void;
  const promise = new Promise<void>(done => { resolve = done; });
  return { promise, resolve };
}

describe('FeatureInvalidationCoordinator', () => {
  it('refetches registered authorized data for live and stale event cursors', async () => {
    const reasons: string[] = [];
    const coordinator = new FeatureInvalidationCoordinator();
    coordinator.register('activity', async request => { reasons.push(`${request.reason}:${request.eventCursor}`); });

    await coordinator.featureChanged('12');
    await coordinator.featureChanged('11');

    expect(reasons).toEqual(['feature_changed:12', 'feature_changed:12']);
  });

  it('coalesces invalidations received during a refetch and does not lose the newest cursor', async () => {
    const first = deferred();
    const cursors: Array<string | undefined> = [];
    const coordinator = new FeatureInvalidationCoordinator();
    coordinator.register('activity', request => {
      cursors.push(request.eventCursor);
      return cursors.length === 1 ? first.promise : undefined;
    });

    const initial = coordinator.featureChanged('40');
    const lost = coordinator.featureChanged('42');
    const stale = coordinator.featureChanged('41');
    expect(cursors).toEqual(['40']);

    first.resolve();
    await Promise.all([initial, lost, stale]);
    expect(cursors).toEqual(['40', '42']);
  });

  it('refetches after each admitted connection so a dropped invalidation converges on reconnect', async () => {
    const refetch = vi.fn();
    const coordinator = new FeatureInvalidationCoordinator();
    coordinator.register('activity', refetch);

    await coordinator.connected('connection-1');
    await coordinator.connected('connection-1');
    await coordinator.connected('connection-2');

    expect(refetch).toHaveBeenCalledTimes(2);
    expect(refetch.mock.calls.map(([request]) => request.reason)).toEqual(['reconnect', 'reconnect']);
  });

  it('refetches on focus, route navigation, and manual refresh', async () => {
    const reasons: string[] = [];
    const coordinator = new FeatureInvalidationCoordinator();
    coordinator.register('features', request => { reasons.push(request.reason); });

    await coordinator.focused();
    await coordinator.routed('/w/one/scenes/two');
    await coordinator.routed('/w/one/scenes/two');
    await coordinator.manualRefresh();

    expect(reasons).toEqual(['focus', 'route', 'manual']);
  });

  it('keeps an invalidation dirty until an authorized consumer registers', async () => {
    const coordinator = new FeatureInvalidationCoordinator();
    await coordinator.featureChanged('51');
    const requests: Array<{ reason: string; eventCursor?: string }> = [];

    const completed = coordinator.register('activity', request => { requests.push(request); });
    await completed.ready;

    expect(requests).toEqual([{ reason: 'feature_changed', eventCursor: '51' }]);
    completed.unregister();
  });

  it('binds socket controls, admitted reconnects, focus, visibility, and route changes', async () => {
    const controls = new Set<(control: { type: string; payload: Record<string, unknown> }) => void>();
    const states = new Set<() => void>();
    let connectionId: string | undefined;
    let route = '/w/one/scenes/one';
    let visible = true;
    const browser = new EventTarget();
    const documentTarget = new EventTarget();
    const reasons: string[] = [];
    const coordinator = new FeatureInvalidationCoordinator();
    coordinator.register('features', request => { reasons.push(`${request.reason}:${request.eventCursor ?? ''}`); });
    const unbind = bindFeatureInvalidation(coordinator, {
      subscribeControl(listener) { controls.add(listener); return () => { controls.delete(listener); }; },
      subscribe(listener) { states.add(listener); return () => { states.delete(listener); }; },
      getSnapshot: () => ({ connectionId }),
    }, {
      browser,
      document: documentTarget,
      isVisible: () => visible,
      route: () => route,
    });

    await vi.waitFor(() => expect(reasons).toEqual(['route:']));
    controls.forEach(listener => listener({ type: 'feature_changed', payload: { eventCursor: '7' } }));
    await vi.waitFor(() => expect(reasons.at(-1)).toBe('feature_changed:7'));
    connectionId = 'admitted-1'; states.forEach(listener => listener());
    await vi.waitFor(() => expect(reasons.at(-1)).toBe('reconnect:7'));
    browser.dispatchEvent(new Event('focus'));
    await vi.waitFor(() => expect(reasons.at(-1)).toBe('focus:7'));
    visible = false; documentTarget.dispatchEvent(new Event('visibilitychange'));
    expect(reasons.at(-1)).toBe('focus:7');
    visible = true; documentTarget.dispatchEvent(new Event('visibilitychange'));
    await vi.waitFor(() => expect(reasons.at(-1)).toBe('focus:7'));
    route = '/w/one/scenes/two'; browser.dispatchEvent(new Event('popstate'));
    await vi.waitFor(() => expect(reasons.at(-1)).toBe('route:7'));

    unbind();
    expect(controls.size).toBe(0);
    expect(states.size).toBe(0);
  });
});
