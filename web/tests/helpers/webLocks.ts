import { vi } from "vitest";

/** Shared lock manager for independently imported modules, like one browser origin. */
export function installWebLocks() {
  const tails = new Map<string, Promise<unknown>>();
  const locks = {
    request(
      name: string,
      optionsOrAction: unknown,
      maybeAction?: (lock: unknown) => unknown,
    ) {
      const action = (maybeAction ?? optionsOrAction) as (
        lock: unknown,
      ) => unknown;
      const prior = tails.get(name);
      if (
        maybeAction &&
        (optionsOrAction as { ifAvailable?: boolean }).ifAvailable &&
        prior
      )
        return Promise.resolve(action(null));
      const operation = (prior ?? Promise.resolve())
        .catch(() => {})
        .then(() => action({ name }));
      tails.set(name, operation);
      const clear = () => {
        if (tails.get(name) === operation) tails.delete(name);
      };
      void operation.then(clear, clear);
      return operation;
    },
  };
  const original = navigator;
  vi.stubGlobal(
    "navigator",
    new Proxy(original, {
      get(target, property) {
        return property === "locks"
          ? locks
          : Reflect.get(target, property, target);
      },
    }),
  );
  return locks;
}
