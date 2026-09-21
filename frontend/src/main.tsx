import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'
import { WasmClient } from './client/wasm'
import { ErrorBoundary } from './components/ErrorBoundary'
import { simStore } from './store/sim'

async function start() {
  const store = simStore.getState()
  await store.init(new WasmClient())
  // A shared link opens paused at the shared moment; otherwise just play.
  if (!(location.hash && (await store.openLink(location.hash)))) store.play()
}
void start()

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ErrorBoundary>
      <App />
    </ErrorBoundary>
  </StrictMode>,
)
