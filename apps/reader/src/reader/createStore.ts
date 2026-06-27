import { useSyncExternalStore } from 'react';

/**
 * Minimal zustand-compatible store factory (the subset the reader uses). Kept local to
 * avoid adding a runtime dependency / touching the root lockfile. Same ergonomics as
 * `zustand`'s `create`: an initializer receiving (set, get), returning a hook that also
 * carries getState / setState / subscribe.
 *
 * Selectors must return stable references for unchanged state (primitives, or objects that
 * only change identity on real updates) so useSyncExternalStore does not loop.
 */
type SetState<T> = (partial: Partial<T> | ((state: T) => Partial<T>)) => void;
type GetState<T> = () => T;
type Initializer<T> = (set: SetState<T>, get: GetState<T>) => T;

export interface StoreHook<T> {
  <U>(selector: (state: T) => U): U;
  getState: GetState<T>;
  setState: SetState<T>;
  subscribe: (listener: () => void) => () => void;
}

export function create<T>(initializer: Initializer<T>): StoreHook<T> {
  let state: T;
  const listeners = new Set<() => void>();

  const setState: SetState<T> = (partial) => {
    const patch = typeof partial === 'function' ? (partial as (s: T) => Partial<T>)(state) : partial;
    state = { ...state, ...patch };
    for (const listener of listeners) listener();
  };
  const getState: GetState<T> = () => state;
  const subscribe = (listener: () => void): (() => void) => {
    listeners.add(listener);
    return () => listeners.delete(listener);
  };

  state = initializer(setState, getState);

  const hook = (<U>(selector: (s: T) => U): U =>
    useSyncExternalStore(
      subscribe,
      () => selector(state),
      () => selector(state),
    )) as StoreHook<T>;

  hook.getState = getState;
  hook.setState = setState;
  hook.subscribe = subscribe;
  return hook;
}
