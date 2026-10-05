import { onUnmounted, ref } from 'vue'

// Subscribes to a project's job events over SSE. onEvent receives
// {type, job_id, job_type, progress, message}; when type is a terminal
// state (done, failed, canceled) the caller should reload project data
// from the API rather than guessing the new state client-side.
export function useSSE(projectId, onEvent) {
  const connected = ref(false)
  let source = null

  function connect() {
    source = new EventSource(`/api/projects/${projectId}/events`)
    source.onopen = () => {
      connected.value = true
    }
    source.onerror = () => {
      connected.value = false
    }
    source.onmessage = (e) => {
      try {
        onEvent(JSON.parse(e.data))
      } catch {
        // malformed or non-JSON event (e.g. a stray comment), ignore it
      }
    }
  }

  function close() {
    source?.close()
    source = null
  }

  connect()
  onUnmounted(close)

  return { connected, close }
}
