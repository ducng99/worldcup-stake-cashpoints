import { createResource, createSignal, For, onMount, Show } from 'solid-js'
import type { User } from '../types'

const storageKey = 'wc-push-settings'

type SavedSettings = {
  userId: number
  notifyLeaderboard: boolean
  notifyMatchStart: boolean
}

async function fetchUsers(): Promise<User[]> {
  const res = await fetch('/api/users')
  if (!res.ok) throw new Error('Failed to fetch users')
  return res.json()
}

function urlBase64ToUint8Array(value: string) {
  const padding = '='.repeat((4 - (value.length % 4)) % 4)
  const base64 = (value + padding).replace(/-/g, '+').replace(/_/g, '/')
  const rawData = window.atob(base64)
  const output = new Uint8Array(rawData.length)
  for (let i = 0; i < rawData.length; i++) output[i] = rawData.charCodeAt(i)
  return output
}

function loadSavedSettings(): SavedSettings {
  const fallback = { userId: 0, notifyLeaderboard: true, notifyMatchStart: true }
  try {
    return { ...fallback, ...JSON.parse(localStorage.getItem(storageKey) || '{}') }
  } catch (_) {
    return fallback
  }
}

export default function Notifications() {
  const [users] = createResource(fetchUsers)
  const saved = loadSavedSettings()
  const [userId, setUserId] = createSignal(saved.userId)
  const [notifyLeaderboard, setNotifyLeaderboard] = createSignal(saved.notifyLeaderboard)
  const [notifyMatchStart, setNotifyMatchStart] = createSignal(saved.notifyMatchStart)
  const [isSubscribed, setIsSubscribed] = createSignal(false)
  const [status, setStatus] = createSignal('')
  const [busy, setBusy] = createSignal(false)

  const supported = () => 'serviceWorker' in navigator && 'PushManager' in window && 'Notification' in window

  onMount(async () => {
    if (!supported()) return
    const registration = await navigator.serviceWorker.register('/sw.js')
    const subscription = await registration.pushManager.getSubscription()
    setIsSubscribed(Boolean(subscription))
  })

  const persistSettings = () => {
    localStorage.setItem(storageKey, JSON.stringify({
      userId: userId(),
      notifyLeaderboard: notifyLeaderboard(),
      notifyMatchStart: notifyMatchStart(),
    }))
  }

  const subscribe = async () => {
    if (!supported()) {
      setStatus('This browser does not support web push notifications.')
      return
    }
    if (!userId()) {
      setStatus('Choose a player account first.')
      return
    }

    setBusy(true)
    setStatus('')
    try {
      const permission = await Notification.requestPermission()
      if (permission !== 'granted') {
        setStatus('Notification permission was not granted.')
        return
      }

      const registration = await navigator.serviceWorker.register('/sw.js')
      const keyRes = await fetch('/api/push/vapid-public-key')
      if (!keyRes.ok) throw new Error('VAPID public key is not configured on the server.')
      const { publicKey } = await keyRes.json()

      const existing = await registration.pushManager.getSubscription()
      const subscription = existing || await registration.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(publicKey),
      })

      const res = await fetch('/api/push/subscriptions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          userId: userId(),
          subscription: subscription.toJSON(),
          notifyLeaderboard: notifyLeaderboard(),
          notifyMatchStart: notifyMatchStart(),
        }),
      })
      if (!res.ok) throw new Error('Failed to save subscription.')

      persistSettings()
      setIsSubscribed(true)
      setStatus('Notifications are enabled for this browser.')
    } catch (err) {
      setStatus(err instanceof Error ? err.message : 'Failed to enable notifications.')
    } finally {
      setBusy(false)
    }
  }

  const updatePreferences = async () => {
    if (!userId()) {
      setStatus('Choose a player account first.')
      return
    }

    setBusy(true)
    setStatus('')
    try {
      const registration = await navigator.serviceWorker.ready
      const subscription = await registration.pushManager.getSubscription()
      if (!subscription) {
        await subscribe()
        return
      }

      const res = await fetch('/api/push/subscriptions/preferences', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          endpoint: subscription.endpoint,
          userId: userId(),
          notifyLeaderboard: notifyLeaderboard(),
          notifyMatchStart: notifyMatchStart(),
        }),
      })
      if (!res.ok) throw new Error('Failed to update preferences.')

      persistSettings()
      setStatus('Notification preferences updated.')
    } catch (err) {
      setStatus(err instanceof Error ? err.message : 'Failed to update preferences.')
    } finally {
      setBusy(false)
    }
  }

  const sendTest = async () => {
    setBusy(true)
    setStatus('')
    try {
      const registration = await navigator.serviceWorker.ready
      const subscription = await registration.pushManager.getSubscription()
      if (!subscription) {
        setStatus('Not subscribed. Enable notifications first.')
        return
      }

      const res = await fetch('/api/push/test', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ subscription: subscription.toJSON() }),
      })
      if (!res.ok) {
        const body = await res.json().catch(() => ({}))
        throw new Error(body.error || 'Failed to send test notification.')
      }
      setStatus('Test notification sent.')
    } catch (err) {
      setStatus(err instanceof Error ? err.message : 'Failed to send test notification.')
    } finally {
      setBusy(false)
    }
  }

  const unsubscribe = async () => {
    setBusy(true)
    setStatus('')
    try {
      const registration = await navigator.serviceWorker.ready
      const subscription = await registration.pushManager.getSubscription()
      if (!subscription) {
        setIsSubscribed(false)
        setStatus('This browser is not subscribed.')
        return
      }

      await fetch('/api/push/subscriptions', {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ endpoint: subscription.endpoint }),
      })
      await subscription.unsubscribe()
      setIsSubscribed(false)
      setStatus('Notifications disabled for this browser.')
    } catch (err) {
      setStatus(err instanceof Error ? err.message : 'Failed to disable notifications.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <section class="notifications-card">
      <div class="section-header">
        <h1 class="section-title">Notifications</h1>
        <div class="section-line" />
      </div>

      <Show when={supported()} fallback={<p class="error">This browser does not support web push notifications.</p>}>
        <Show when={users.loading}><p class="loading">Loading players...</p></Show>
        <Show when={users.error}><p class="error">Failed to load players.</p></Show>
        <Show when={users()}>
          <div class="settings-panel">
            <label class="settings-label">
              Player account
              <select class="player-filter settings-select" value={userId()} onChange={(event) => setUserId(Number(event.currentTarget.value))}>
                <option value="0">Choose player</option>
                <For each={users()}>{(user) => <option value={user.id}>{user.name}</option>}</For>
              </select>
            </label>

            <label class="checkbox-row">
              <input type="checkbox" checked={notifyLeaderboard()} onChange={(event) => setNotifyLeaderboard(event.currentTarget.checked)} />
              Leaderboard position changes
            </label>
            <label class="checkbox-row">
              <input type="checkbox" checked={notifyMatchStart()} onChange={(event) => setNotifyMatchStart(event.currentTarget.checked)} />
              My matches starting
            </label>

            <div class="settings-actions">
              <button class="primary-button" disabled={busy()} onClick={isSubscribed() ? updatePreferences : subscribe}>
                {isSubscribed() ? 'Save Preferences' : 'Enable Notifications'}
              </button>
              <Show when={isSubscribed()}>
                <button class="secondary-button" disabled={busy()} onClick={sendTest}>Send Test</button>
                <button class="secondary-button" disabled={busy()} onClick={unsubscribe}>Unsubscribe</button>
              </Show>
            </div>

            <Show when={status()}><p class="settings-status">{status()}</p></Show>
          </div>
        </Show>
      </Show>
    </section>
  )
}
