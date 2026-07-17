import test from 'node:test'; import assert from 'node:assert/strict'; import { readFile } from 'node:fs/promises'
test('LabFlow has sample batch, schedule and quality data', async()=>{const source=await readFile(new URL('../src/main.js',import.meta.url),'utf8'); assert.match(source,/今日样本批次队列/); assert.match(source,/实验员排班/); assert.match(source,/质控任务/); assert.match(source,/LB-0716-082/)})

test('LabFlow binds real API actions while keeping a demo fallback', async()=>{const source=await readFile(new URL('../src/main.js',import.meta.url),'utf8'); assert.match(source,/createApiClient/); assert.match(source,/data-action="checkin"/); assert.match(source,/data-action="status"/); assert.match(source,/data-action="complete-followup"/); assert.match(source,/refreshFromApi/); assert.match(source,/演示数据/)})

test('Vite proxies the default API path to the local Go service', async()=>{const source=await readFile(new URL('../vite.config.js',import.meta.url),'utf8'); assert.match(source,/server/); assert.match(source,/proxy/); assert.match(source,/localhost:8080/)})

test('admin UI exposes sample detail, report review and fictional-data states', async()=>{const source=await readFile(new URL('../src/main.js',import.meta.url),'utf8'); assert.match(source,/样本送检/); assert.match(source,/待复核/); assert.match(source,/报告结果/); assert.match(source,/listSamples/); assert.match(source,/getSample/); assert.match(source,/演示数据/)})

test('admin sample workspace has filters, selected detail timeline and editable report form', async()=>{const source=await readFile(new URL('../src/main.js',import.meta.url),'utf8'); assert.match(source,/data-sample-status/); assert.match(source,/data-sample-keyword/); assert.match(source,/selectedSample/); assert.match(source,/事件时间线/); assert.match(source,/<textarea/); assert.match(source,/report-result/); assert.match(source,/report-remark/)})
