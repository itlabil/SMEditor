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

// nomor is the 1-based segment number shown on screen.
export function updateSegment(id, nomor, segment) {
  return request('PUT', `/projects/${id}/highlight/segmen/${nomor}`, segment)
}

export function deleteSegment(id, nomor) {
  return request('DELETE', `/projects/${id}/highlight/segmen/${nomor}`)
}

// draft: { tim_a: { nama, pick: [], ban: [] }, tim_b: { ... } }
export function updateDraft(id, draft) {
  return request('PUT', `/projects/${id}/highlight/draft`, draft)
}
