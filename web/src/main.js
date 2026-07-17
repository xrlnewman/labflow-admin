import './styles.css'
import { createApiClient } from './api.js'

const api = createApiClient()

const demoAppointments = [
  { id: 'LB-0716-082', patient: '样本批次 A082', department: '生化检验', doctor: '林实验员', scheduledAt: '2026-07-16T09:30:00+08:00', status: '检测排队' },
  { id: 'LB-0716-081', patient: '样本批次 A081', department: '微生物检验', doctor: '沈实验员', scheduledAt: '2026-07-16T09:45:00+08:00', status: '已收样' },
  { id: 'LB-0716-080', patient: '样本批次 A080', department: '免疫检验', doctor: '赵实验员', scheduledAt: '2026-07-16T10:00:00+08:00', status: '已完成' },
  { id: 'LB-0716-079', patient: '样本批次 A079', department: '生化检验', doctor: '林实验员', scheduledAt: '2026-07-16T10:15:00+08:00', status: '待收样' },
  { id: 'LB-0716-078', patient: '样本批次 A078', department: '分子诊断', doctor: '周实验员', scheduledAt: '2026-07-16T10:30:00+08:00', status: '待收样' },
]

const demoFollowups = [
  { id: 'QC-0716-012', patient: '样本批次 A082', summary: '质控曲线复核', dueAt: '今天 16:00', status: '待完成' },
  { id: 'QC-0716-011', patient: '样本批次 A081', summary: '试剂批号复核', dueAt: '今天 17:30', status: '待完成' },
  { id: 'QC-0716-010', patient: '样本批次 A080', summary: '检测结果双人复核', dueAt: '明天 09:30', status: '待完成' },
  { id: 'QC-0715-009', patient: '样本批次 A079', summary: '设备校准记录归档', dueAt: '已完成', status: '已完成' },
]

const demoSamples = [
  { id: 'SM-0717-001', subjectAlias: '受检者-001', sampleType: '血液', collectedAt: '2026-07-17T08:00:00Z', status: '待送检', tests: [{ name: '血常规' }] },
  { id: 'SM-0717-002', subjectAlias: '受检者-002', sampleType: '尿液', collectedAt: '2026-07-17T09:00:00Z', status: '已接收', tests: [{ name: '尿常规' }] },
  { id: 'SM-0717-003', subjectAlias: '受检者-003', sampleType: '咽拭子', collectedAt: '2026-07-17T09:30:00Z', status: '检验中', tests: [{ name: '病原筛查' }] },
  { id: 'SM-0717-004', subjectAlias: '受检者-004', sampleType: '血液', collectedAt: '2026-07-17T10:00:00Z', status: '待复核', tests: [{ name: '生化组合' }], report: { result: '阴性', remark: '需双人复核' } },
]

const demoDashboard = { todayAppointments: 86, averageWaitMinutes: 12, completed: 58, checkedIn: 42, pendingFollowups: 12 }
const statusColors = { 待收样: 'coral', 已收样: 'indigo', 检测排队: 'amber', 检测中: 'green', 已完成: 'green', 已作废: 'gray' }
const nav = [
  ['overview', '运营总览', '⌂'],
  ['samples', '样本送检', '◌'],
  ['queue', '样本批次队列', '▤'],
  ['doctors', '实验员排班', '◉'],
  ['patients', '样本档案', '♧'],
  ['followups', '质控任务', '✓'],
  ['mobile', '移动端体验', '⌁'],
]

let appointments = demoAppointments.map((item) => ({ ...item }))
let followupTasks = demoFollowups.map((item) => ({ ...item }))
let samples = demoSamples.map((item) => ({ ...item }))
let dashboard = { ...demoDashboard }
let page = 'overview'
let toast = ''
let toastTimer
let dataSource = '演示数据'
let isSyncing = false

function timeLabel(value) {
  const match = String(value ?? '').match(/T(\d{2}:\d{2})/)
  return match?.[1] || String(value ?? '').slice(0, 5) || '--:--'
}

function normalizeAppointment(item) {
  return {
    id: item.id,
    patientId: item.patientId,
    patient: item.patient || '未命名样本批次',
    department: item.department || '待分检验线',
    doctor: item.doctor || '待安排实验员',
    scheduledAt: item.scheduledAt || '',
    status: item.status || '待收样',
  }
}

function normalizeFollowup(item) {
  return {
    id: item.id,
    patientId: item.patientId,
    patient: item.patient || '未命名样本批次',
    summary: item.summary || '检测质控任务',
    dueAt: item.dueAt || '--',
    status: item.status || '待完成',
  }
}

function normalizeSample(item) {
  return { ...item, subjectAlias: item.subjectAlias || '未命名受检者', sampleType: item.sampleType || '未分类', status: item.status || '待送检', tests: Array.isArray(item.tests) ? item.tests : [] }
}

function showToast(message) {
  toast = message
  render()
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => {
    toast = ''
    render()
  }, 2200)
}

function appointmentAction(appointment) {
  if (appointment.status === '待收样') return `<button class="text-action" data-action="checkin" data-appointment-id="${appointment.id}">确认收样</button>`
  if (appointment.status === '已收样') return `<button class="text-action" data-action="status" data-next-status="检测排队" data-appointment-id="${appointment.id}">加入检测队列</button>`
  if (appointment.status === '检测排队') return `<button class="text-action" data-action="status" data-next-status="检测中" data-appointment-id="${appointment.id}">开始检测</button>`
  if (appointment.status === '检测中') return `<button class="text-action" data-action="status" data-next-status="已完成" data-appointment-id="${appointment.id}">完成检测</button>`
  return '<button class="text-action" data-toast="该样本批次已完成，无需重复操作">查看详情</button>'
}

function header(title) {
  return `<header><span>工作台　/　<strong>${title}</strong></span><span class="header-tools"><span>2026 年 7 月 16 日</span><span class="data-source ${dataSource === 'API 数据' ? 'remote' : ''}">● ${isSyncing ? '同步中' : dataSource}</span><button class="refresh" data-refresh ${isSyncing ? 'disabled' : ''}>↻ 刷新</button></span></header>`
}

function render() {
  const title = nav.find((item) => item[0] === page)?.[1] || '运营总览'
  const content = page === 'overview' ? overview() : page === 'samples' ? sampleQueue() : page === 'queue' ? queue() : page === 'doctors' ? doctors() : page === 'patients' ? patients() : page === 'followups' ? followups() : mobileView()
  document.querySelector('#app').innerHTML = `<div class="shell"><aside><div class="brand"><span>✚</span><div><strong>LabFlow</strong><small>实验室运营中心</small></div></div><div class="clinic">● 上海静安联合实验室　⌄</div><p class="caption">样本运营</p><nav>${nav.map((item) => `<button class="${page === item[0] ? 'active' : ''}" data-page="${item[0]}"><i>${item[2]}</i>${item[1]}${item[0] === 'queue' ? '<em>8</em>' : ''}</button>`).join('')}</nav><div class="user"><b>许</b><span><strong>许汝林</strong><small>运营管理员</small></span></div></aside><main>${header(title)}<section class="heading"><div><p>THURSDAY, JUL 16 · LABFLOW</p><h1>${title} <i>✦</i></h1><label>让每一个样本批次，都有清晰可追踪的下一步。</label></div><button class="primary" data-action="create-appointment">＋ 新建样本批次</button></section>${content}<footer>LabFlow 医疗样本检测与质控 · 免费开源 · 演示数据不含真实样本隐私</footer><div class="toast" ${toast ? '' : 'hidden'}>${toast}</div></main></div>`
  bind()
}

function overview() {
  return `<section class="metrics"><article class="metric dark"><span>今日样本批次</span><strong>${dashboard.todayAppointments}</strong><small>↗ 较昨日 +14.6%</small></article><article class="metric"><span>平均检测周转</span><strong>${dashboard.averageWaitMinutes}<small> 分钟</small></strong><small class="good">较上周 -3 分钟</small></article><article class="metric"><span>今日完成</span><strong>${dashboard.completed}<small> 批次</small></strong><div class="progress"><i style="width:68%"></i></div></article><article class="metric warm"><span>待质控</span><strong>${dashboard.pendingFollowups}<small> 条</small></strong><small class="coral">今日需完成</small></article></section><section class="grid"><article class="panel calendar"><div class="panel-head"><div><h2>今日样本批次队列</h2><p>7 月 16 日 · 周四 · 共 ${dashboard.todayAppointments} 个样本批次</p></div><button class="link" data-page="queue">查看队列 →</button></div><div class="timeline">${appointments.slice(0, 4).map((appointment) => `<div class="time-row"><span>${timeLabel(appointment.scheduledAt)}</span><i class="time-dot ${statusColors[appointment.status] || 'indigo'}"></i><div><strong>${appointment.patient}</strong><small>${appointment.department} · ${appointment.status}</small></div><b class="status ${statusColors[appointment.status] || 'indigo'}">${appointment.status}</b></div>`).join('')}</div></article><article class="panel"><div class="panel-head"><div><h2>检验线负载</h2><p>当前时段设备利用率</p></div><button class="link" data-page="doctors">排班管理 →</button></div><div class="load-list">${[['生化检验', '32 / 40', '80%', 'indigo'], ['微生物检验', '18 / 24', '75%', 'coral'], ['免疫检验', '12 / 18', '67%', 'green'], ['分子诊断', '8 / 12', '66%', 'amber']].map((item) => `<div class="load"><div><strong>${item[0]}</strong><span>${item[1]}</span></div><div class="load-bar"><i class="${item[3]}" style="width:${item[2]}"></i></div><b>${item[2]}</b></div>`).join('')}</div></article></section><section class="grid lower"><article class="panel"><div class="panel-head"><div><h2>质控完成趋势</h2><p>近 7 日任务完成率</p></div><span class="legend">本周平均 84%</span></div><div class="spark"><i style="height:38%"></i><i style="height:58%"></i><i style="height:46%"></i><i style="height:74%"></i><i style="height:66%"></i><i style="height:88%"></i><i class="today" style="height:80%"></i></div><div class="days"><span>周五</span><span>周六</span><span>周日</span><span>周一</span><span>周二</span><span>周三</span><span>今天</span></div></article><article class="panel tasks"><div class="panel-head"><div><h2>待办提醒</h2><p>需要运营人员跟进的事项</p></div></div><div class="task"><span class="task-icon coral">!</span><div><strong>3 个样本批次需要补充信息</strong><small>样本批次队列 · 10 分钟前</small></div><button data-page="queue">处理</button></div><div class="task"><span class="task-icon amber">✓</span><div><strong>${dashboard.pendingFollowups} 条质控今日到期</strong><small>实验室质控 · 32 分钟前</small></div><button data-page="followups">查看</button></div></article></section>`
}

function queue() {
  return `<section class="panel full"><div class="panel-head"><div><h2>样本批次队列</h2><p>${dataSource === 'API 数据' ? 'API 实时样本批次' : '20 条演示样本批次'} · 支持收样、排队、检测和完成</p></div><span class="chip">今天　⌄</span></div><div class="table"><div class="th"><span>批次编号 / 样本</span><span>检验线</span><span>时间</span><span>状态</span><span>操作</span></div>${appointments.concat(dataSource === 'API 数据' ? [] : appointments.slice(0, 3)).map((appointment) => `<div class="tr"><span><strong>${appointment.id}</strong><small>${appointment.patient}</small></span><span>${appointment.department}</span><span>${timeLabel(appointment.scheduledAt)}</span><b class="status ${statusColors[appointment.status] || 'indigo'}">${appointment.status}</b><span>${appointmentAction(appointment)}</span></div>`).join('')}</div></section>`
}

function sampleAction(sample) {
  if (sample.status === '待送检') return `<button class="text-action" data-action="receive-sample" data-sample-id="${sample.id}">接收样本</button>`
  if (sample.status === '已接收') return `<button class="text-action" data-action="start-sample-test" data-sample-id="${sample.id}">开始检验</button>`
  if (sample.status === '检验中') return `<button class="text-action" data-action="report-sample" data-sample-id="${sample.id}">提交报告</button>`
  if (sample.status === '待复核') return `<button class="text-action" data-action="review-sample" data-sample-id="${sample.id}">复核报告</button>`
  if (sample.status === '已出报告') return `<button class="text-action" data-action="archive-sample" data-sample-id="${sample.id}">归档报告</button>`
  return '<span class="muted-action">已完成闭环</span>'
}

function sampleQueue() {
  return `<section class="panel full"><div class="panel-head"><div><h2>样本送检</h2><p>${dataSource === 'API 数据' ? 'API 实时样本' : '4 条虚构样本'} · 送检、检验、报告和归档全流程</p></div><span class="chip">虚构数据 / 演示流程</span></div><div class="table"><div class="th"><span>样本编号 / 受检者</span><span>样本类型</span><span>检验项目</span><span>状态</span><span>操作</span></div>${samples.length ? samples.map((sample) => `<div class="tr"><span><strong>${sample.id}</strong><small>${sample.subjectAlias}</small></span><span>${sample.sampleType}</span><span>${sample.tests.map((item) => item.name).join('、') || '待配置'}</span><b class="status ${sample.status === '已归档' ? 'green' : sample.status === '待复核' ? 'amber' : 'indigo'}">${sample.status}</b><span>${sampleAction(sample)}</span></div>`).join('') : '<div class="empty">暂无样本，点击新建样本送检</div>'}</div><div class="sample-note">报告结果仅用于演示；样本详情包含完整事件时间线。</div></section>`
}

function doctors() {
  return `<section class="panel full"><div class="panel-head"><div><h2>实验员排班</h2><p>8 位实验员 · 今日 42 个可检测时段</p></div><button class="primary small" data-toast="排班编辑器已打开">编辑排班</button></div><div class="doctor-grid">${[['林实验员', '生化检验', '32 批检测中', 'indigo'], ['沈实验员', '微生物检验', '18 批待复核', 'coral'], ['赵实验员', '免疫检验', '检测中', 'green'], ['周实验员', '分子诊断', '8 批排队', 'amber'], ['陈实验员', '生化检验', '午间休息', 'gray'], ['王实验员', '微生物检验', '6 批排队', 'indigo']].map((doctor) => `<article><div class="doctor-avatar ${doctor[3]}">${doctor[0][0]}</div><div><strong>${doctor[0]}</strong><small>${doctor[1]}</small></div><span>${doctor[2]}</span><div class="schedule-line"><i style="width:78%"></i></div></article>`).join('')}</div></section>`
}

function patients() {
  return `<section class="panel full"><div class="panel-head"><div><h2>样本档案</h2><p>30 条虚构批次档案 · 仅用于界面演示</p></div><button class="link" data-toast="导出任务已创建">导出列表 ↓</button></div><div class="table"><div class="th"><span>样本批次 / 编号</span><span>检验线</span><span>最近检测</span><span>质控状态</span><span>操作</span></div>${[['样本批次 A082', 'LB-2038', '生化检验', '07/16', '待质控'], ['样本批次 A081', 'LB-2037', '微生物检验', '07/15', '进行中'], ['样本批次 A080', 'LB-2036', '免疫检验', '07/14', '已完成'], ['样本批次 A079', 'LB-2035', '生化检验', '07/13', '待质控'], ['样本批次 A078', 'LB-2034', '分子诊断', '07/12', '已完成']].map((sample) => `<div class="tr"><span><strong>${sample[0]}</strong><small>${sample[1]}</small></span><span>${sample[2]}</span><span>${sample[3]}</span><b class="status ${sample[4] === '已完成' ? 'green' : 'coral'}">${sample[4]}</b><button class="text-action" data-toast="${sample[0]} 档案已打开">查看档案</button></div>`).join('')}</div></section>`
}

function followups() {
  return `<section class="panel full"><div class="panel-head"><div><h2>质控任务</h2><p>${dataSource === 'API 数据' ? 'API 实时质控' : '12 条待跟进任务'} · 由实验员复核后记录</p></div><span class="chip">全部任务　⌄</span></div><div class="follow-list">${followupTasks.map((item) => `<article><span class="task-icon ${item.status === '已完成' ? 'green' : 'coral'}">✓</span><div><strong>${item.id} · ${item.patient}</strong><p>${item.summary}</p><small>${item.dueAt} · ${dataSource === 'API 数据' ? 'API 数据' : '演示任务'}</small></div>${item.status === '已完成' ? '<button class="text-action" data-toast="该质控已经完成">查看</button>' : `<button class="text-action" data-action="complete-followup" data-followup-id="${item.id}">完成任务</button>`}</article>`).join('')}</div></section>`
}

function mobileView() {
  return `<section class="mobile-panel"><div class="mobile-panel__hero"><span>LABFLOW MOBILE</span><h2>我的检测与质控</h2><p>样本端可在同一套闭环 API 中完成收样、排队、检测和质控确认。</p><button class="primary" data-action="create-appointment">＋ 创建演示样本批次</button></div><div class="mobile-list"><h3>今日样本批次</h3>${appointments.slice(0, 4).map((appointment) => `<article class="mobile-card"><div><small>${timeLabel(appointment.scheduledAt)} · ${appointment.department}</small><strong>${appointment.patient}</strong><span>${appointment.doctor} · ${appointment.status}</span></div><b class="status ${statusColors[appointment.status] || 'indigo'}">${appointment.status}</b>${appointmentAction(appointment)}</article>`).join('')}</div><div class="mobile-list"><h3>我的质控</h3>${followupTasks.slice(0, 3).map((item) => `<article class="mobile-card"><div><small>${item.dueAt}</small><strong>${item.summary}</strong><span>${item.patient} · ${item.status}</span></div>${item.status === '已完成' ? '<b class="status green">已完成</b>' : `<button class="text-action" data-action="complete-followup" data-followup-id="${item.id}">完成质控</button>`}</article>`).join('')}</div></section>`
}

async function refreshFromApi({ quiet = false } = {}) {
  if (isSyncing) return
  isSyncing = true
  render()
  try {
    const [nextDashboard, nextAppointments, nextFollowups, nextSamples] = await Promise.all([
      api.getDashboard(),
      api.listAppointments({ page: 1, pageSize: 20 }),
      api.listFollowups({ page: 1, pageSize: 20 }),
      api.listSamples({ page: 1, pageSize: 20 }),
    ])
    dashboard = { ...demoDashboard, ...nextDashboard }
    appointments = (nextAppointments?.list || []).map(normalizeAppointment)
    followupTasks = (nextFollowups?.list || []).map(normalizeFollowup)
    samples = (nextSamples?.list || []).map(normalizeSample)
    dataSource = 'API 数据'
    if (!quiet) toast = '已从 LabFlow API 刷新样本数据'
  } catch (error) {
    dataSource = '演示数据'
    if (!quiet) toast = `API 暂不可用，继续使用演示数据：${error.message}`
  } finally {
    isSyncing = false
    render()
  }
}

async function sampleActionHandler(button) {
  const id = button.dataset.sampleId
  const sample = samples.find((item) => item.id === id)
  if (!sample) return
  try {
    let updated
    if (button.dataset.action === 'receive-sample') updated = await api.receiveSample(id, '收样员')
    if (button.dataset.action === 'start-sample-test') updated = await api.startSampleTest(id, '检验员')
    if (button.dataset.action === 'report-sample') updated = await api.reportSample(id, { result: '阴性', remark: '演示报告，等待复核' })
    if (button.dataset.action === 'review-sample') updated = await api.reviewSample(id, '审核员')
    if (button.dataset.action === 'archive-sample') updated = await api.archiveSample(id, '归档员')
    if (updated) {
      const detail = await api.getSample(id)
      samples = samples.map((item) => item.id === id ? normalizeSample(detail || updated) : item)
      dataSource = 'API 数据'
      showToast(`${sample.subjectAlias} 已更新为${updated.status}`)
    }
  } catch (error) {
    showToast(`样本操作失败：${error.message}`)
  }
}

function replaceAppointment(updated) {
  appointments = appointments.map((item) => item.id === updated.id ? normalizeAppointment(updated) : item)
}

async function advanceAppointment(button) {
  const id = button.dataset.appointmentId
  const appointment = appointments.find((item) => item.id === id)
  if (!appointment) return
  const nextStatus = button.dataset.nextStatus
  try {
    const updated = button.dataset.action === 'checkin'
      ? await api.checkinAppointment(id)
      : await api.updateAppointmentStatus(id, nextStatus, '运营人员')
    replaceAppointment(updated)
    dataSource = 'API 数据'
    showToast(`${appointment.patient} 已更新为${updated.status}`)
  } catch (error) {
    dataSource = '演示数据'
    showToast(`接口暂不可用，已保留演示数据：${error.message}`)
  }
}

async function completeFollowup(button) {
  const id = button.dataset.followupId
  const task = followupTasks.find((item) => item.id === id)
  if (!task) return
  try {
    const updated = await api.completeFollowup(id)
    followupTasks = followupTasks.map((item) => item.id === id ? normalizeFollowup(updated) : item)
    dataSource = 'API 数据'
    showToast(`${task.patient} 的质控已完成`)
  } catch (error) {
    dataSource = '演示数据'
    showToast(`接口暂不可用，已保留演示任务：${error.message}`)
  }
}

async function createAppointment() {
  try {
    const created = await api.createAppointment({ patient: '移动端演示样本批次', patientId: 'LB-MOBILE-DEMO', department: '生化检验', doctor: '林实验员', scheduledAt: new Date().toISOString() })
    appointments = [normalizeAppointment(created), ...appointments]
    dataSource = 'API 数据'
    showToast('样本批次已创建，可继续在移动端确认收样')
  } catch (error) {
    dataSource = '演示数据'
    showToast(`API 暂不可用，保留演示样本批次：${error.message}`)
  }
}

function bind() {
  document.querySelectorAll('[data-page]').forEach((element) => element.addEventListener('click', () => {
    page = element.dataset.page
    render()
  }))
  document.querySelectorAll('[data-toast]').forEach((element) => element.addEventListener('click', () => showToast(element.dataset.toast)))
  document.querySelectorAll('[data-refresh]').forEach((element) => element.addEventListener('click', () => refreshFromApi()))
  document.querySelectorAll('[data-action]').forEach((element) => element.addEventListener('click', () => {
    if (element.dataset.action === 'checkin' || element.dataset.action === 'status') return advanceAppointment(element)
    if (element.dataset.action === 'complete-followup') return completeFollowup(element)
    if (['receive-sample', 'start-sample-test', 'report-sample', 'review-sample', 'archive-sample'].includes(element.dataset.action)) return sampleActionHandler(element)
    if (element.dataset.action === 'create-appointment') return createAppointment()
    return undefined
  }))
}

render()
refreshFromApi({ quiet: true })
