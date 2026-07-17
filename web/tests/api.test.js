import test from 'node:test'
import assert from 'node:assert/strict'

import { createApiClient } from '../src/api.js'

function response(data, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    async json() {
      return { code: 0, message: 'ok', data }
    },
  }
}

test('defaults to /api/v1 and adds an idempotency key to writes', async () => {
  const requests = []
  const client = createApiClient({
    fetchImpl: async (url, init) => {
      requests.push({ url, init })
      return response({ id: 'LB-1', status: '已收样' })
    },
  })

  const appointment = await client.checkinAppointment('LB-1')

  assert.equal(appointment.id, 'LB-1')
  assert.equal(requests[0].url, '/api/v1/appointments/LB-1/checkin')
  assert.equal(requests[0].init.method, 'POST')
  assert.match(requests[0].init.headers['Idempotency-Key'], /^cf-/)
})

test('uses a configured API origin without duplicating the API path', async () => {
  const requests = []
  const client = createApiClient({
    baseUrl: 'http://localhost:8080/api/v1/',
    fetchImpl: async (url) => {
      requests.push(url)
      return response({ list: [], total: 0 })
    },
  })

  await client.listAppointments({ page: 1, pageSize: 20 })

  assert.equal(requests[0], 'http://localhost:8080/api/v1/appointments?page=1&pageSize=20')
})

test('rejects non-zero API envelopes so callers can keep demo data', async () => {
  const client = createApiClient({
    fetchImpl: async () => ({
      ok: false,
      status: 409,
      async json() {
        return { code: 409, message: '状态不可推进', data: null }
      },
    }),
  })

  await assert.rejects(() => client.updateAppointmentStatus('LB-1', '检测排队'), /状态不可推进/)
})

test('exposes mobile lifecycle and follow-up operations through the same client', async () => {
  const paths = []
  const client = createApiClient({
    fetchImpl: async (url) => {
      paths.push(url)
      return response({ id: 'ok' })
    },
  })

  await client.createAppointment({ patient: '样本批次 A001', department: '生化检验' })
  await client.checkinAppointment('LB-1')
  await client.updateAppointmentStatus('LB-1', '检测排队')
  await client.updateAppointmentStatus('LB-1', '检测中')
  await client.updateAppointmentStatus('LB-1', '已完成')
  await client.completeFollowup('FW-1')

  assert.deepEqual(paths, [
    '/api/v1/appointments',
    '/api/v1/appointments/LB-1/checkin',
    '/api/v1/appointments/LB-1/status',
    '/api/v1/appointments/LB-1/status',
    '/api/v1/appointments/LB-1/status',
    '/api/v1/followups/FW-1/complete',
  ])
})

test('exposes sample report lifecycle with query filters and idempotency', async () => {
  const requests = []
  const client = createApiClient({
    fetchImpl: async (url, init) => {
      requests.push({ url, init })
      return response({ id: 'SM-1', status: '已接收', events: [] })
    },
  })

  await client.listSamples({ page: 1, pageSize: 20, status: '待复核', keyword: '受检者' })
  await client.getSample('SM-1')
  await client.createSample({ subjectAlias: '受检者-01', sampleType: '血液', tests: ['血常规'] }, 'sample-key')
  await client.receiveSample('SM-1', '收样员', 'receive-key')
  await client.startSampleTest('SM-1', '检验员')
  await client.reportSample('SM-1', { result: '阴性', remark: '稳定' })
  await client.reviewSample('SM-1', '审核员')
  await client.archiveSample('SM-1', '归档员')

  assert.equal(requests[0].url, '/api/v1/samples?page=1&pageSize=20&status=%E5%BE%85%E5%A4%8D%E6%A0%B8&keyword=%E5%8F%97%E6%A3%80%E8%80%85')
  assert.equal(requests[2].init.headers['Idempotency-Key'], 'sample-key')
  assert.equal(requests[3].url, '/api/v1/samples/SM-1/receive')
  assert.equal(requests[4].url, '/api/v1/samples/SM-1/start-test')
  assert.equal(requests[5].url, '/api/v1/samples/SM-1/report')
  assert.equal(requests[6].url, '/api/v1/samples/SM-1/review')
  assert.equal(requests[7].url, '/api/v1/samples/SM-1/archive')
})
