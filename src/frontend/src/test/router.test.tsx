import { act, renderHook } from '@testing-library/react';
import { expect, it } from 'vitest';

import { navigate, useLocation } from '@/lib/router';

it('observes query-only navigation for notification thread deep links', () => {
  window.history.replaceState({}, '', '/w/workspace/scenes/scene?thread=one');
  const { result } = renderHook(() => useLocation());
  expect(result.current.search).toBe('?thread=one');
  act(() => navigate('/w/workspace/scenes/scene?thread=two'));
  expect(result.current.search).toBe('?thread=two');
});
