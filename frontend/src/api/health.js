import { request } from './client'

export function checkHealth() {
  return request('GET', '/health')
}
