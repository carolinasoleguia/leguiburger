import { ref } from "vue";
export const soundEnabled = ref(false);
export const soundError = ref("");
let context;
export async function toggleOrderSound() {
  if (soundEnabled.value) {
    soundEnabled.value = false;
    return;
  }
  try {
    const Audio = window.AudioContext || window.webkitAudioContext;
    if (!Audio) throw new Error("Audio unavailable");
    context ||= new Audio();
    await context.resume();
    soundEnabled.value = context.state === "running";
    soundError.value = soundEnabled.value
      ? ""
      : "El navegador no habilitó el sonido. Volvé a intentar.";
  } catch {
    soundError.value = "No pudimos activar el sonido en este navegador.";
  }
}
export function notifyOrderChange(previous, next) {
  if (
    !previous ||
    !next ||
    previous.id !== next.id ||
    (previous.status === next.status &&
      String(previous.total_paid) === String(next.total_paid))
  )
    return;
  if (!soundEnabled.value || document.hidden || context?.state !== "running")
    return;
  try {
    const start = context.currentTime;
    for (const [offset, frequency] of [
      [0, 660],
      [0.14, 880],
    ]) {
      const oscillator = context.createOscillator(),
        gain = context.createGain();
      oscillator.type = "sine";
      oscillator.frequency.value = frequency;
      gain.gain.setValueAtTime(0, start + offset);
      gain.gain.linearRampToValueAtTime(0.12, start + offset + 0.015);
      gain.gain.exponentialRampToValueAtTime(0.001, start + offset + 0.22);
      oscillator.connect(gain);
      gain.connect(context.destination);
      oscillator.start(start + offset);
      oscillator.stop(start + offset + 0.24);
      oscillator.onended = () => {
        oscillator.disconnect();
        gain.disconnect();
      };
    }
  } catch {
    /* Audio must never prevent a verified order update. */
  }
}
