import { ref, watch, onBeforeUnmount } from "vue";
// fetch streaming keeps the capability in Authorization, out of URLs and logs.
export function useOrderStream({ path, token, enabled, refresh }) {
  const live = ref(false);
  let stopped = false,
    controller,
    timer,
    attempts = 0,
    running = false,
    refreshing = false,
    again = false;
  async function sync() {
    if (refreshing) {
      again = true;
      return;
    }
    refreshing = true;
    try {
      do {
        again = false;
        await refresh();
      } while (again && !stopped);
    } finally {
      refreshing = false;
    }
  }
  async function connect() {
    if (running || stopped || document.hidden || !enabled() || !token()) return;
    running = true;
    controller = new AbortController();
    let watchdog = setTimeout(() => controller.abort(), 45000);
    try {
      const response = await fetch(path, {
        credentials: "omit",
        cache: "no-store",
        referrerPolicy: "no-referrer",
        headers: {
          Authorization: `Bearer ${token()}`,
          Accept: "text/event-stream",
        },
        signal: controller.signal,
      });
      if (
        !response.ok ||
        !response.headers.get("content-type")?.includes("text/event-stream")
      )
        throw new Error("Stream unavailable");
      const reader = response.body.getReader(),
        decoder = new TextDecoder();
      let buffer = "";
      try {
        while (!stopped) {
          const { value, done } = await reader.read();
          if (done) break;
          clearTimeout(watchdog);
          watchdog = setTimeout(() => controller.abort(), 45000);
          buffer += decoder
            .decode(value, { stream: true })
            .replaceAll("\r", "");
          if (buffer.length > 32768) throw new Error("Invalid stream");
          let end;
          while ((end = buffer.indexOf("\n\n")) >= 0) {
            const frame = buffer.slice(0, end);
            buffer = buffer.slice(end + 2);
            if (/^event: (ready|changed)$/m.test(frame)) {
              live.value = true;
              attempts = 0;
              await sync();
            }
          }
        }
      } finally {
        await reader.cancel().catch(() => {});
      }
    } catch {
      /* Normal fallback: polling remains available. */
    } finally {
      clearTimeout(watchdog);
      running = false;
      live.value = false;
      if (!stopped && enabled() && !document.hidden)
        timer = setTimeout(
          connect,
          Math.min(30000, 1000 * 2 ** Math.min(attempts++, 5)) +
            Math.random() * 500,
        );
    }
  }
  function visibility() {
    clearTimeout(timer);
    if (document.hidden || !enabled()) {
      controller?.abort();
      live.value = false;
    } else {
      sync();
      connect();
    }
  }
  document.addEventListener("visibilitychange", visibility);
  const unwatch = watch([enabled, token], visibility, {
    immediate: true,
  });
  let lastSync = Date.now();
  const fallback = setInterval(() => {
    if (
      !document.hidden &&
      enabled() &&
      Date.now() - lastSync >= (live.value ? 120000 : 30000)
    ) {
      lastSync = Date.now();
      sync();
    }
  }, 5000);
  onBeforeUnmount(() => {
    stopped = true;
    unwatch();
    clearInterval(fallback);
    clearTimeout(timer);
    controller?.abort();
    document.removeEventListener("visibilitychange", visibility);
  });
  return { live };
}
