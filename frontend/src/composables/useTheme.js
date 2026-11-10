import { ref } from 'vue';

const theme = ref(localStorage.getItem('legui-theme') === 'light' ? 'light' : 'dark');
document.documentElement.dataset.theme = theme.value;

export function useTheme() {
  function toggleTheme() {
    theme.value = theme.value === 'dark' ? 'light' : 'dark';
    document.documentElement.dataset.theme = theme.value;
    localStorage.setItem('legui-theme', theme.value);
  }
  return { theme, toggleTheme };
}
