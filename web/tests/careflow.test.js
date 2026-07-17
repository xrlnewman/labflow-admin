import test from 'node:test'; import assert from 'node:assert/strict'; import { readFile } from 'node:fs/promises'
test('LabFlow has sample batch, schedule and quality data', async()=>{const source=await readFile(new URL('../src/main.js',import.meta.url),'utf8'); assert.match(source,/今日样本批次队列/); assert.match(source,/实验员排班/); assert.match(source,/质控任务/); assert.match(source,/LB-0716-082/)})

test('LabFlow binds real API actions while keeping a demo fallback', async()=>{const source=await readFile(new URL('../src/main.js',import.meta.url),'utf8'); assert.match(source,/createApiClient/); assert.match(source,/data-action="checkin"/); assert.match(source,/data-action="status"/); assert.match(source,/data-action="complete-followup"/); assert.match(source,/refreshFromApi/); assert.match(source,/演示数据/)})

test('Vite proxies the default API path to the local Go service', async()=>{const source=await readFile(new URL('../vite.config.js',import.meta.url),'utf8'); assert.match(source,/server/); assert.match(source,/proxy/); assert.match(source,/localhost:8080/)})
