import type {
  Collaborator,
  ExcalidrawImperativeAPI,
  SocketId,
} from '@excalidraw/excalidraw/types';

import type { Presence } from './presence';

// These native API subscriptions only touch ephemeral appState, never scene content.
export function bindNativePresence(
  presence: Presence,
  api: ExcalidrawImperativeAPI,
  captureUpdate: Parameters<ExcalidrawImperativeAPI['updateScene']>[0]['captureUpdate'],
) {
  let active = true;
  let pendingNativeFollow: { target: string | null } | undefined;
  let nativeFollowScheduled = false;

  const project = () => {
    const state = presence.getSnapshot();
    const collaborators = new Map<SocketId, Collaborator>();
    for (const participant of state.participants) {
      const cursor = state.cursors[participant.connectionId];
      collaborators.set(participant.connectionId as SocketId, {
        id: participant.connectionId,
        socketId: participant.connectionId as SocketId,
        username: participant.displayName,
        isCurrentUser: participant.connectionId === state.connectionId,
        isInCall: participant.voiceJoined,
        isMuted: participant.voiceMuted,
        ...(cursor?.cursor
          ? { pointer: { ...cursor.cursor, tool: 'pointer' as const } }
          : {}),
        selectedElementIds: Object.fromEntries(
          (cursor?.selectedIds ?? []).map((id) => [id, true]),
        ),
      });
    }

    const target = state.participants.find(
      (participant) => participant.connectionId === state.following,
    );
    const targetId = target?.connectionId ?? null;
    const nativeFollow = api.getAppState().userToFollow;
    const nativeTargetId = nativeFollow?.socketId ?? null;
    const appState: {
      followedBy: Set<SocketId>;
      userToFollow: { socketId: SocketId; username: string } | null;
    } = {
      followedBy: new Set(
        state.participants
          .filter((participant) => participant.following === state.connectionId)
          .map((participant) => participant.connectionId as SocketId),
      ),
      userToFollow: nativeFollow,
    };
    // Excalidraw emits onUserFollow whenever this object reference changes.
    // Project only a server target transition so roster refreshes cannot echo
    // UNFOLLOW/FOLLOW back to the room and oscillate indefinitely.
    if (targetId !== nativeTargetId) {
      appState.userToFollow = target
        ? {
            socketId: target.connectionId as SocketId,
            username: target.displayName,
          }
        : null;
    }
    api.updateScene({ collaborators, appState, captureUpdate });
  };

  const onNativeFollow = ({
    action,
    userToFollow,
  }: Parameters<Parameters<ExcalidrawImperativeAPI['onUserFollow']>[0]>[0]) => {
    pendingNativeFollow = {
      target: action === 'FOLLOW' ? userToFollow.socketId : null,
    };
    if (nativeFollowScheduled) return;
    nativeFollowScheduled = true;
    queueMicrotask(() => {
      nativeFollowScheduled = false;
      const pending = pendingNativeFollow;
      pendingNativeFollow = undefined;
      if (
        active &&
        pending &&
        presence.getSnapshot().following !== pending.target
      ) {
        presence.follow(pending.target);
      }
    });
  };

  const cleanups = [
    presence.subscribe(project),
    presence.subscribeViewport((viewport) =>
      api.updateScene({
        appState: {
          scrollX: viewport.scrollX,
          scrollY: viewport.scrollY,
          zoom: { value: viewport.zoom as never },
        },
        captureUpdate,
      }),
    ),
    api.onScrollChange((scrollX, scrollY, zoom) =>
      presence.viewport({ scrollX, scrollY, zoom: zoom.value }),
    ),
    api.onUserFollow(onNativeFollow),
    api.onChange((_elements, state) =>
      presence.selection(
        Object.keys(state.selectedElementIds).filter(
          (id) => state.selectedElementIds[id],
        ),
      ),
    ),
  ];
  const state = api.getAppState();
  presence.viewport({
    scrollX: state.scrollX,
    scrollY: state.scrollY,
    zoom: state.zoom.value,
  });
  project();

  return () => {
    active = false;
    pendingNativeFollow = undefined;
    for (const cleanup of cleanups) cleanup();
    api.updateScene({
      collaborators: new Map(),
      appState: { userToFollow: null, followedBy: new Set() },
      captureUpdate,
    });
  };
}
