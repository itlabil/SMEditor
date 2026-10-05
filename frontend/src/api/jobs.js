import { request } from './client'

export function cancelJob(id) {
  return request('POST', `/jobs/${id}/cancel`)
}
