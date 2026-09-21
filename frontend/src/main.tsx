import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'
import { WasmClient } from './client/wasm'
import { ErrorBoundary } from './components/ErrorBoundary'
import { simStore } from './store/sim'

void simStore
  .getState()
  .init(new WasmClient())
  .then(() => simStore.getState().play())

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <ErrorBoundary>
      <App />
    </ErrorBoundary>
  </StrictMode>,
)
