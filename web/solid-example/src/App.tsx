import { createSignal, onMount, onCleanup, createResource, Show, For } from 'solid-js'
import solidLogo from './assets/solid.svg'
import viteLogo from './assets/vite.svg'
import heroImg from './assets/hero.png'
import { fetchCurrentUser, setupEventsWebSocket } from './api'
import type { User } from './api'
import './App.css'

function App() {
  const [user] = createResource<User | null>(fetchCurrentUser)
  const [events, setEvents] = createSignal<any[]>([])
  const [wsStatus, setWsStatus] = createSignal<'connecting' | 'connected' | 'disconnected'>('connecting')

  onMount(() => {
    const ws = setupEventsWebSocket((data) => {
      setEvents((prev) => [data, ...prev].slice(0, 5))
    })
    
    ws.onopen = () => setWsStatus('connected')
    ws.onclose = () => setWsStatus('disconnected')

    onCleanup(() => ws.close())
  })

  return (
    <div class="app-container">
      <header class="glass-header">
        <div class="logo-group">
          <img src={viteLogo} class="logo vite" alt="Vite logo" />
          <img src={solidLogo} class="logo solid" alt="Solid logo" />
        </div>
        <h1>Go Modulith + SolidJS</h1>
        <div class="user-badge">
          <Show when={user()} fallback={<span class="anon">Guest Mode</span>}>
            {(u) => <span class="user-name">{u().name}</span>}
          </Show>
          <div class={`status-indicator ${wsStatus()}`} title={`WebSocket: ${wsStatus()}`} />
        </div>
      </header>

      <main>
        <section class="hero-section">
          <div class="hero-image-container">
            <img src={heroImg} class="hero-image" alt="Modulith Architecture" />
            <div class="hero-overlay" />
          </div>
          <div class="hero-content">
            <h2>Seamless Integration</h2>
            <p>
              Connect your SolidJS frontend to a powerful Go backend with gRPC-Gateway, 
              GraphQL, and real-time WebSockets.
            </p>
            <div class="action-buttons">
              <a href="https://github.com/LoopContext/go-modulith-template" class="btn primary">View Template</a>
              <button class="btn secondary" onClick={() => window.open('/docs/SOLIDJS_INTEGRATION.md')}>Read Guide</button>
            </div>
          </div>
        </section>

        <section class="features-grid">
          <div class="card glass">
            <h3>gRPC Gateway (REST)</h3>
            <p>Lightweight JSON communication translated directly from your Protobuf definitions.</p>
            <div class="code-snippet">
              <code>fetch('/v1/auth/me')</code>
            </div>
          </div>
          <div class="card glass">
            <h3>GraphQL</h3>
            <p>Flexible data fetching with built-in subscriptions and deep hierarchy support.</p>
            <div class="code-snippet">
              <code>query {'{'} me {'{'} name {'}'} {'}'}</code>
            </div>
          </div>
          <div class="card glass">
            <h3>Real-time Bus</h3>
            <p>Live events pushed directly to your UI via the internal Event Bus and WebSockets.</p>
            <div class="event-feed">
              <Show when={events().length > 0} fallback={<p class="placeholder">Awaiting live events...</p>}>
                <For each={events()}>
                  {(event) => (
                    <div class="event-item">
                      <span class="type">{event.type}</span>
                      <span class="time">{new Date().toLocaleTimeString()}</span>
                    </div>
                  )}
                </For>
              </Show>
            </div>
          </div>
        </section>
      </main>

      <footer>
        <p>Built with Go Modulith Template & SolidJS</p>
      </footer>
    </div>
  )
}

export default App
