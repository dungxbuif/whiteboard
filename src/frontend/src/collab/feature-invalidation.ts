export type FeatureInvalidationReason = 'feature_changed' | 'reconnect' | 'focus' | 'route' | 'manual';

export type FeatureRefetchRequest = {
  reason: FeatureInvalidationReason;
  eventCursor?: string;
};

export type FeatureRefetch = (request: FeatureRefetchRequest) => void | Promise<void>;

type Control = { type: string; payload: Record<string, unknown> };
type InvalidationSource = {
  subscribeControl: (listener: (control: Control) => void) => () => void;
  subscribe: (listener: () => void) => () => void;
  getSnapshot: () => { connectionId?: string };
};
type InvalidationEnvironment = {
  browser: EventTarget;
  document: EventTarget;
  isVisible: () => boolean;
  route: () => string;
  onError?: (error: unknown) => void;
};

type Registration = {
  ready: Promise<void>;
  unregister: () => void;
};

export class FeatureInvalidationCoordinator {
  private readonly consumers = new Map<string, FeatureRefetch>();
  private pending?: FeatureRefetchRequest;
  private running?: Promise<void>;
  private highestCursor?: bigint;
  private highestCursorText?: string;
  private connectionId?: string;
  private route?: string;

  register(key: string, refetch: FeatureRefetch): Registration {
    this.consumers.set(key, refetch);
    const ready = this.flush();
    return {
      ready,
      unregister: () => {
        if (this.consumers.get(key) === refetch) this.consumers.delete(key);
      },
    };
  }

  featureChanged(eventCursor: string): Promise<void> {
    if (!/^[1-9]\d*$/.test(eventCursor)) return Promise.reject(new Error('event cursor must be a positive decimal string'));
    const cursor = BigInt(eventCursor);
    if (this.highestCursor === undefined || cursor > this.highestCursor) {
      this.highestCursor = cursor;
      this.highestCursorText = eventCursor;
    }
    return this.invalidate({ reason: 'feature_changed', eventCursor: this.highestCursorText });
  }

  connected(connectionId: string): Promise<void> {
    if (!connectionId || connectionId === this.connectionId) return this.running ?? Promise.resolve();
    this.connectionId = connectionId;
    return this.invalidate({ reason: 'reconnect', eventCursor: this.highestCursorText });
  }

  focused(): Promise<void> {
    return this.invalidate({ reason: 'focus', eventCursor: this.highestCursorText });
  }

  routed(route: string): Promise<void> {
    if (route === this.route) return this.running ?? Promise.resolve();
    this.route = route;
    return this.invalidate({ reason: 'route', eventCursor: this.highestCursorText });
  }

  manualRefresh(): Promise<void> {
    return this.invalidate({ reason: 'manual', eventCursor: this.highestCursorText });
  }

  private invalidate(request: FeatureRefetchRequest): Promise<void> {
    this.pending = request;
    return this.flush();
  }

  private flush(): Promise<void> {
    if (this.running) return this.running;
    if (!this.pending || this.consumers.size === 0) return Promise.resolve();
    this.running = this.drain().finally(() => { this.running = undefined; });
    return this.running;
  }

  private async drain(): Promise<void> {
    while (this.pending && this.consumers.size > 0) {
      const request = this.pending;
      this.pending = undefined;
      await Promise.all([...this.consumers.values()].map(refetch => refetch(request)));
    }
  }
}

export function bindFeatureInvalidation(
  coordinator: FeatureInvalidationCoordinator,
  source: InvalidationSource,
  environment: InvalidationEnvironment,
): () => void {
  const run = (operation: Promise<void>) => { void operation.catch(error => environment.onError?.(error)); };
  const control = (message: Control) => {
    if (message.type === 'feature_changed' && typeof message.payload.eventCursor === 'string') {
      run(coordinator.featureChanged(message.payload.eventCursor));
    }
  };
  const admitted = () => {
    const { connectionId } = source.getSnapshot();
    if (connectionId) run(coordinator.connected(connectionId));
  };
  const focus = () => { run(coordinator.focused()); };
  const visible = () => { if (environment.isVisible()) run(coordinator.focused()); };
  const route = () => { run(coordinator.routed(environment.route())); };
  const unsubscribeControl = source.subscribeControl(control);
  const unsubscribeState = source.subscribe(admitted);
  environment.browser.addEventListener('focus', focus);
  environment.browser.addEventListener('popstate', route);
  environment.document.addEventListener('visibilitychange', visible);
  admitted();
  route();
  return () => {
    unsubscribeControl();
    unsubscribeState();
    environment.browser.removeEventListener('focus', focus);
    environment.browser.removeEventListener('popstate', route);
    environment.document.removeEventListener('visibilitychange', visible);
  };
}
