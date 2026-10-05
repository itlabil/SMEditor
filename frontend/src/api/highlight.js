import { request } from './client'

export function saveHighlight(id, body) {
  return request('POST', `/projects/${id}/highlight`, { body })
}

export function getHighlight(id) {
  return request('GET', `/projects/${id}/highlight`)
}

export function deleteHighlight(id) {
  return request('DELETE', `/projects/${id}/highlight`)
}

export function narasiUrl(id) {
  return `/api/projects/${id}/narasi`
}
