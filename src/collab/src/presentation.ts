import { randomUUID } from 'node:crypto';
import { CollabError, revision } from './auth.js';

export const PRESENTATION_WAITING_LIMIT = 100;
export const PRESENTATION_AUDIENCE_LIMIT = 100;
export const PRESENTATION_REACTIONS = ['👏', '❤️', '👍', '🎉', '😂'] as const;
export const PRESENTATION_REACTION_TTL_MS = 3_000;
export const PRESENTATION_REACTION_RATE_MS = 1_000;

export type PresentationParticipant = { connectionId: string; principalId: string; displayName: string };
export type PresentationReaction = { id: string; connectionId: string; emoji: typeof PRESENTATION_REACTIONS[number]; expiresAt: number };
export type PresentationState = {
  room: string; sessionId: string; hostConnectionId: string; hostPrincipalId: string; revision: number;
  pending: Map<string, PresentationParticipant>; admitted: Map<string, PresentationParticipant>;
  frameId: string | null; spotlight: string | null; locked: boolean;
  reactions: PresentationReaction[]; lastReaction: Map<string, number>;
};

const sessionID = (value: unknown): value is string => typeof value === 'string' && /^[0-9a-f-]{36}$/.test(value);
const connectionID = (value: unknown): value is string => typeof value === 'string' && value.length > 0 && value.length <= 128;
const elementID = (value: unknown): value is string => typeof value === 'string' && value.length > 0 && value.length <= 128;

export class PresentationSessions {
  private sessions = new Map<string, PresentationState>();
  constructor(private now: () => number = Date.now) {}

  start(room: string, id: string, hostConnectionId: string, hostPrincipalId: string) {
    if (!sessionID(id) || !connectionID(hostConnectionId) || !hostPrincipalId) throw new CollabError('PRESENTATION_INVALID', 'Invalid presentation session', 400);
    if (this.sessions.has(room)) throw new CollabError('PRESENTATION_ACTIVE', 'A presentation is already active', 409);
    const state: PresentationState = { room, sessionId: id, hostConnectionId, hostPrincipalId, revision: 1, pending: new Map(), admitted: new Map(), frameId: null, spotlight: null, locked: false, reactions: [], lastReaction: new Map() };
    this.sessions.set(room, state); return this.snapshot(state);
  }

  wait(room: string, id: string, participant: PresentationParticipant) {
    const state = this.session(room, id);
    if (participant.connectionId === state.hostConnectionId || !connectionID(participant.connectionId) || !participant.principalId || !participant.displayName) throw new CollabError('PRESENTATION_INVALID', 'Invalid waiting participant', 400);
    if (!state.pending.has(participant.connectionId) && state.pending.size >= PRESENTATION_WAITING_LIMIT) throw new CollabError('PRESENTATION_FULL', 'Waiting room is full', 409);
    state.pending.set(participant.connectionId, { ...participant }); return { sessionId: id, status: 'pending' as const, connectionId: participant.connectionId };
  }

  admit(room: string, id: string, hostConnectionId: string, targetConnectionId: string, expectedRevision: number) {
    const state = this.host(room, id, hostConnectionId, expectedRevision), participant = state.pending.get(targetConnectionId);
    if (!participant) throw new CollabError('PRESENTATION_TARGET', 'Waiting participant is unavailable', 404);
    if (state.admitted.size >= PRESENTATION_AUDIENCE_LIMIT) throw new CollabError('PRESENTATION_FULL', 'Presentation audience is full', 409);
    state.pending.delete(targetConnectionId); state.admitted.set(targetConnectionId, participant); state.revision++;
    return { state: this.snapshot(state), participant: { ...participant } };
  }

  remove(room: string, id: string, hostConnectionId: string, targetConnectionId: string, expectedRevision: number) {
    const state = this.host(room, id, hostConnectionId, expectedRevision);
    if (!state.pending.delete(targetConnectionId) && !state.admitted.delete(targetConnectionId)) throw new CollabError('PRESENTATION_TARGET', 'Participant is unavailable', 404);
    if (state.spotlight === targetConnectionId) state.spotlight = null;
    state.lastReaction.delete(targetConnectionId); state.revision++; return this.snapshot(state);
  }

  setFrame(room: string, id: string, hostConnectionId: string, frameId: string | null, expectedRevision: number) {
    const state = this.host(room, id, hostConnectionId, expectedRevision);
    if (frameId !== null && !elementID(frameId)) throw new CollabError('PRESENTATION_INVALID', 'Invalid frame ID', 400);
    state.frameId = frameId; state.revision++; return this.snapshot(state);
  }

  spotlight(room: string, id: string, hostConnectionId: string, targetConnectionId: string | null, expectedRevision: number) {
    const state = this.host(room, id, hostConnectionId, expectedRevision);
    if (targetConnectionId !== null && targetConnectionId !== state.hostConnectionId && !state.admitted.has(targetConnectionId)) throw new CollabError('PRESENTATION_TARGET', 'Spotlight participant is unavailable', 404);
    state.spotlight = targetConnectionId; state.revision++; return this.snapshot(state);
  }

  setLocked(room: string, id: string, hostConnectionId: string, locked: boolean, expectedRevision: number) {
    const state = this.host(room, id, hostConnectionId, expectedRevision);
    state.locked = locked; state.revision++; return this.snapshot(state);
  }

  transfer(room: string, id: string, hostConnectionId: string, targetConnectionId: string, targetPrincipalId: string, expectedRevision: number) {
    const state = this.host(room, id, hostConnectionId, expectedRevision);
    if (!connectionID(targetConnectionId) || !targetPrincipalId) throw new CollabError('PRESENTATION_TARGET', 'Transfer target is unavailable', 404);
    state.hostConnectionId = targetConnectionId; state.hostPrincipalId = targetPrincipalId;
    if (state.spotlight === hostConnectionId) state.spotlight = null;
    state.revision++; return this.snapshot(state);
  }

  react(room: string, id: string, connectionId: string, emoji: string) {
    const state = this.session(room, id);
    if (connectionId !== state.hostConnectionId && !state.admitted.has(connectionId)) throw new CollabError('PRESENTATION_FORBIDDEN', 'Only admitted participants may react', 403);
    if (!(PRESENTATION_REACTIONS as readonly string[]).includes(emoji)) throw new CollabError('PRESENTATION_REACTION', 'Reaction is not allowed', 400);
    const now = this.now(), last = state.lastReaction.get(connectionId);
    if (last !== undefined && now - last < PRESENTATION_REACTION_RATE_MS) throw new CollabError('PRESENTATION_REACTION_RATE', 'Reaction rate exceeded', 429);
    state.lastReaction.set(connectionId, now); state.reactions = state.reactions.filter(item => item.expiresAt > now);
    const reaction: PresentationReaction = { id: randomUUID(), connectionId, emoji: emoji as PresentationReaction['emoji'], expiresAt: now + PRESENTATION_REACTION_TTL_MS };
    state.reactions.push(reaction); if (state.reactions.length > PRESENTATION_AUDIENCE_LIMIT) state.reactions.splice(0, state.reactions.length - PRESENTATION_AUDIENCE_LIMIT); return { ...reaction };
  }

  state(room: string) { const state = this.sessions.get(room); return state ? this.snapshot(state) : undefined; }
  active(room: string, id: string) { const state = this.sessions.get(room); return !!state && state.sessionId === id; }

  disconnect(connectionId: string) {
    for (const [room, state] of this.sessions) {
      if (state.hostConnectionId === connectionId) { this.sessions.delete(room); return { ended: this.snapshot(state), participant: false }; }
      if (state.pending.delete(connectionId) || state.admitted.delete(connectionId)) {
        if (state.spotlight === connectionId) state.spotlight = null;
        state.lastReaction.delete(connectionId); state.revision++;
        return { state: this.snapshot(state), participant: true };
      }
    }
    return undefined;
  }

  stop(room: string, id: string, hostConnectionId: string) {
    const state = this.host(room, id, hostConnectionId, this.sessions.get(room)?.revision ?? 0);
    this.sessions.delete(room); return this.snapshot(state);
  }

  private session(room: string, id: string) {
    const state = this.sessions.get(room);
    if (!state || state.sessionId !== id) throw new CollabError('PRESENTATION_ENDED', 'Presentation has ended', 409);
    return state;
  }
  private host(room: string, id: string, hostConnectionId: string, expectedRevision: number) {
    const state = this.session(room, id);
    if (state.hostConnectionId !== hostConnectionId) throw new CollabError('PRESENTATION_FORBIDDEN', 'Only the current host may control presentation', 403);
    if (!Number.isInteger(expectedRevision) || expectedRevision !== state.revision) throw new CollabError('PRESENTATION_CONFLICT', 'Presentation revision changed', 409);
    return state;
  }
  private snapshot(state: PresentationState) {
    const now = this.now(); state.reactions = state.reactions.filter(item => item.expiresAt > now);
    return { sessionId: state.sessionId, hostConnectionId: state.hostConnectionId, hostPrincipalId: state.hostPrincipalId, revision: state.revision, pending: [...state.pending.values()].map(value => ({ ...value })), admitted: [...state.admitted.values()].map(value => ({ ...value })), frameId: state.frameId, spotlight: state.spotlight, locked: state.locked, reactions: state.reactions.map(value => ({ ...value })) };
  }
}

function object(value: unknown) { if (!value || typeof value !== 'object' || Array.isArray(value)) throw new CollabError('PRESENTATION_INVALID', 'Invalid presentation control', 400); return value as Record<string, unknown>; }
function strict(value: unknown, keys: string[]) { const input = object(value); if (Object.keys(input).some(key => !keys.includes(key))) throw new CollabError('PRESENTATION_INVALID', 'Invalid presentation control', 400); return input; }
function revisioned(value: unknown, keys: string[]) {
  const input = strict(value, ['sessionId', 'expectedRevision', ...keys]), expected = Number(input.expectedRevision);
  if (!sessionID(input.sessionId) || !revision(input.expectedRevision) || input.expectedRevision === '0' || !Number.isSafeInteger(expected)) throw new CollabError('PRESENTATION_INVALID', 'Invalid presentation control', 400);
  return { input, sessionId: input.sessionId, expectedRevision: expected };
}
export function parsePresentationStart(value: unknown) { const input = strict(value, ['token']); if (typeof input.token !== 'string' || input.token.length > 16_384) throw new CollabError('PRESENTATION_INVALID', 'Invalid presentation control', 400); return { token: input.token }; }
export function parsePresentationAdmit(value: unknown) { const input = strict(value, ['connectionId', 'token', 'expectedRevision']), expected = Number(input.expectedRevision); if (!connectionID(input.connectionId) || typeof input.token !== 'string' || input.token.length > 16_384 || !revision(input.expectedRevision) || input.expectedRevision === '0' || !Number.isSafeInteger(expected)) throw new CollabError('PRESENTATION_INVALID', 'Invalid presentation control', 400); return { connectionId: input.connectionId, token: input.token, expectedRevision: expected }; }
export function parsePresentationSetFrame(value: unknown) { const { input, sessionId, expectedRevision } = revisioned(value, ['frameId']); if (input.frameId !== null && !elementID(input.frameId)) throw new CollabError('PRESENTATION_INVALID', 'Invalid frame ID', 400); return { sessionId, expectedRevision, frameId: input.frameId as string | null }; }
export function parsePresentationSpotlight(value: unknown) { const { input, sessionId, expectedRevision } = revisioned(value, ['connectionId']); if (input.connectionId !== null && !connectionID(input.connectionId)) throw new CollabError('PRESENTATION_INVALID', 'Invalid spotlight target', 400); return { sessionId, expectedRevision, connectionId: input.connectionId as string | null }; }
export function parsePresentationLock(value: unknown) { const { input, sessionId, expectedRevision } = revisioned(value, ['locked']); if (typeof input.locked !== 'boolean') throw new CollabError('PRESENTATION_INVALID', 'Invalid presentation lock', 400); return { sessionId, expectedRevision, locked: input.locked }; }
export function parsePresentationTarget(value: unknown) { const { input, sessionId, expectedRevision } = revisioned(value, ['connectionId']); if (!connectionID(input.connectionId)) throw new CollabError('PRESENTATION_INVALID', 'Invalid participant target', 400); return { sessionId, expectedRevision, connectionId: input.connectionId }; }
export function parsePresentationReaction(value: unknown) { const input = strict(value, ['sessionId', 'emoji']); if (!sessionID(input.sessionId) || typeof input.emoji !== 'string' || !(PRESENTATION_REACTIONS as readonly string[]).includes(input.emoji)) throw new CollabError('PRESENTATION_REACTION', 'Reaction is not allowed', 400); return { sessionId: input.sessionId, emoji: input.emoji }; }
export function parsePresentationLeave(value: unknown) { const input = strict(value, ['sessionId']); if (!sessionID(input.sessionId)) throw new CollabError('PRESENTATION_INVALID', 'Invalid presentation session', 400); return { sessionId: input.sessionId }; }
