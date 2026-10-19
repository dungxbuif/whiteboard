// Collaboration state belongs to each admitted provider, so scenes and tabs cannot
// share microphone/peer state through a process-wide Socket.IO singleton.
export { Presence } from '@/collab/presence';
export type { Participant, PresenceState, Cursor } from '@/collab/presence';
export { Voice } from '@/collab/voice';
export type { VoiceState } from '@/collab/voice';
