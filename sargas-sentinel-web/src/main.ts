import { mount } from 'svelte'
import './app.css'
import App from './App.svelte'

try {
  const guardado = localStorage.getItem('sargas.tema');
  if (guardado === 'neon-blue-light') {
    document.documentElement.dataset.theme = guardado;
  }
} catch {
  // El tema oscuro sigue disponible sin almacenamiento local.
}

const app = mount(App, {
  target: document.getElementById('app')!,
})

export default app
