export class ApiError extends Error {
  constructor(error, status) {
    super(error?.message || 'Terjadi kesalahan')
    this.code = error?.code
    this.details = error?.details
    this.status = status
  }
}

export async function request(method, path, body) {
  const res = await fetch(`/api${path}`, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  })
  if (res.status === 204) return null
  const json = await res.json()
  if (!res.ok) throw new ApiError(json.error, res.status)
  return json.data
}
