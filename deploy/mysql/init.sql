-- LabFlow synthetic operational data only. Never load real medical records here.
CREATE TABLE IF NOT EXISTS departments (
  id VARCHAR(64) PRIMARY KEY,
  name VARCHAR(64) NOT NULL UNIQUE
);
CREATE TABLE IF NOT EXISTS doctors (
  id VARCHAR(64) PRIMARY KEY,
  name VARCHAR(64) NOT NULL,
  department VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  today_count INT NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS patients (
  id VARCHAR(64) PRIMARY KEY,
  name VARCHAR(64) NOT NULL,
  phone VARCHAR(32) NOT NULL,
  last_visit VARCHAR(32) NOT NULL,
  created_at VARCHAR(64) NOT NULL
);
CREATE TABLE IF NOT EXISTS appointments (
  id VARCHAR(64) PRIMARY KEY,
  patient_id VARCHAR(64) NOT NULL,
  patient_name VARCHAR(64) NOT NULL,
  department VARCHAR(64) NOT NULL,
  doctor VARCHAR(64) NOT NULL,
  scheduled_at VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  INDEX idx_appointments_status_time (status, scheduled_at)
);
CREATE TABLE IF NOT EXISTS appointment_events (
  id VARCHAR(64) PRIMARY KEY,
  appointment_id VARCHAR(64) NOT NULL,
  from_status VARCHAR(32) NOT NULL,
  to_status VARCHAR(32) NOT NULL,
  actor VARCHAR(64) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  INDEX idx_appointment_events_appointment (appointment_id, created_at)
);
CREATE TABLE IF NOT EXISTS followups (
  id VARCHAR(64) PRIMARY KEY,
  patient_id VARCHAR(64) NOT NULL,
  patient_name VARCHAR(64) NOT NULL,
  summary VARCHAR(255) NOT NULL,
  due_at VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  INDEX idx_followups_status_due (status, due_at)
);
CREATE TABLE IF NOT EXISTS samples (
  id VARCHAR(64) PRIMARY KEY,
  subject_alias VARCHAR(128) NOT NULL,
  sample_type VARCHAR(64) NOT NULL,
  collected_at VARCHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  INDEX idx_samples_status_updated (status, updated_at),
  INDEX idx_samples_subject_alias (subject_alias)
);
CREATE TABLE IF NOT EXISTS sample_tests (
  id VARCHAR(64) PRIMARY KEY,
  sample_id VARCHAR(64) NOT NULL,
  name VARCHAR(128) NOT NULL,
  status VARCHAR(32) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  INDEX idx_sample_tests_sample (sample_id, id),
  CONSTRAINT fk_sample_tests_sample FOREIGN KEY (sample_id) REFERENCES samples(id)
);
CREATE TABLE IF NOT EXISTS sample_reports (
  id VARCHAR(64) PRIMARY KEY,
  sample_id VARCHAR(64) NOT NULL,
  result VARCHAR(255) NOT NULL,
  remark VARCHAR(500) NOT NULL,
  status VARCHAR(32) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  updated_at VARCHAR(64) NOT NULL,
  INDEX idx_sample_reports_sample (sample_id, created_at)
);
CREATE TABLE IF NOT EXISTS sample_events (
  id VARCHAR(64) PRIMARY KEY,
  sample_id VARCHAR(64) NOT NULL,
  action VARCHAR(64) NOT NULL,
  from_status VARCHAR(32) NOT NULL,
  to_status VARCHAR(32) NOT NULL,
  actor VARCHAR(64) NOT NULL,
  created_at VARCHAR(64) NOT NULL,
  INDEX idx_sample_events_sample (sample_id, created_at)
);

INSERT IGNORE INTO departments (id,name) VALUES
 ('line-biochem','生化检验'),('line-micro','微生物检验'),('line-immuno','免疫检验'),('line-molecular','分子诊断');
INSERT IGNORE INTO doctors (id,name,department,status,today_count) VALUES
 ('tech-01','林实验员','生化检验','检测中',18),('tech-02','沈实验员','微生物检验','检测中',16),
 ('tech-03','赵实验员','免疫检验','检测中',12),('tech-04','周实验员','分子诊断','休息中',10),
 ('tech-05','陈实验员','生化检验','检测中',14),('tech-06','王实验员','微生物检验','检测中',16);
INSERT IGNORE INTO patients (id,name,phone,last_visit,created_at) VALUES
 ('PT-001','演示样本01','13800000001','2026-07-15','2026-07-01'),('PT-002','演示样本02','13800000002','2026-07-15','2026-07-01'),
 ('PT-003','演示样本03','13800000003','2026-07-14','2026-07-01'),('PT-004','演示样本04','13800000004','2026-07-14','2026-07-01'),
 ('PT-005','演示样本05','13800000005','2026-07-13','2026-07-01'),('PT-006','演示样本06','13800000006','2026-07-13','2026-07-01'),
 ('PT-007','演示样本07','13800000007','2026-07-12','2026-07-01'),('PT-008','演示样本08','13800000008','2026-07-12','2026-07-01'),
 ('PT-009','演示样本09','13800000009','2026-07-11','2026-07-01'),('PT-010','演示样本10','13800000010','2026-07-11','2026-07-01'),
 ('PT-011','演示样本11','13800000011','2026-07-10','2026-07-01'),('PT-012','演示样本12','13800000012','2026-07-10','2026-07-01'),
 ('PT-013','演示样本13','13800000013','2026-07-09','2026-07-01'),('PT-014','演示样本14','13800000014','2026-07-09','2026-07-01'),
 ('PT-015','演示样本15','13800000015','2026-07-08','2026-07-01'),('PT-016','演示样本16','13800000016','2026-07-08','2026-07-01'),
 ('PT-017','演示样本17','13800000017','2026-07-07','2026-07-01'),('PT-018','演示样本18','13800000018','2026-07-07','2026-07-01'),
 ('PT-019','演示样本19','13800000019','2026-07-06','2026-07-01'),('PT-020','演示样本20','13800000020','2026-07-06','2026-07-01'),
 ('PT-021','演示样本21','13800000021','2026-07-05','2026-07-01'),('PT-022','演示样本22','13800000022','2026-07-05','2026-07-01'),
 ('PT-023','演示样本23','13800000023','2026-07-04','2026-07-01'),('PT-024','演示样本24','13800000024','2026-07-04','2026-07-01'),
 ('PT-025','演示样本25','13800000025','2026-07-03','2026-07-01'),('PT-026','演示样本26','13800000026','2026-07-03','2026-07-01'),
 ('PT-027','演示样本27','13800000027','2026-07-02','2026-07-01'),('PT-028','演示样本28','13800000028','2026-07-02','2026-07-01'),
 ('PT-029','演示样本29','13800000029','2026-07-01','2026-07-01'),('PT-030','演示样本30','13800000030','2026-07-01','2026-07-01');
INSERT IGNORE INTO appointments (id,patient_id,patient_name,department,doctor,scheduled_at,status,created_at,updated_at) VALUES
 ('LB-0716-081','PT-001','样本批次 A081','生化检验','林实验员','2026-07-16T08:00:00+08:00','已完成','2026-07-16T00:00:00Z','2026-07-16T01:00:00Z'),
 ('LB-0716-082','PT-002','样本批次 A082','微生物检验','沈实验员','2026-07-16T09:00:00+08:00','检测中','2026-07-16T00:00:00Z','2026-07-16T01:00:00Z'),
 ('LB-0716-083','PT-003','样本批次 A083','免疫检验','赵实验员','2026-07-16T10:00:00+08:00','检测排队','2026-07-16T00:00:00Z','2026-07-16T01:00:00Z'),
 ('LB-0716-084','PT-004','样本批次 A084','分子诊断','周实验员','2026-07-16T11:00:00+08:00','已收样','2026-07-16T00:00:00Z','2026-07-16T01:00:00Z'),
 ('LB-0716-085','PT-005','样本批次 A085','生化检验','陈实验员','2026-07-16T12:00:00+08:00','待收样','2026-07-16T00:00:00Z','2026-07-16T01:00:00Z'),
 ('LB-0716-086','PT-006','样本批次 A086','微生物检验','王实验员','2026-07-16T13:00:00+08:00','已完成','2026-07-16T00:00:00Z','2026-07-16T01:00:00Z'),
 ('LB-0716-087','PT-007','样本批次 A087','免疫检验','赵实验员','2026-07-16T14:00:00+08:00','检测中','2026-07-16T00:00:00Z','2026-07-16T01:00:00Z'),
 ('LB-0716-088','PT-008','样本批次 A088','分子诊断','周实验员','2026-07-16T15:00:00+08:00','检测排队','2026-07-16T00:00:00Z','2026-07-16T01:00:00Z'),
 ('LB-0716-089','PT-009','样本批次 A089','生化检验','林实验员','2026-07-16T16:00:00+08:00','已收样','2026-07-16T00:00:00Z','2026-07-16T01:00:00Z'),
 ('LB-0716-090','PT-010','样本批次 A090','微生物检验','沈实验员','2026-07-16T17:00:00+08:00','待收样','2026-07-16T00:00:00Z','2026-07-16T01:00:00Z'),
 ('LB-0716-091','PT-011','样本批次 A091','免疫检验','赵实验员','2026-07-16T08:30:00+08:00','已完成','2026-07-16T00:00:00Z','2026-07-16T01:00:00Z'),
 ('LB-0716-092','PT-012','样本批次 A092','分子诊断','周实验员','2026-07-16T09:30:00+08:00','检测排队','2026-07-16T00:00:00Z','2026-07-16T01:00:00Z');
INSERT IGNORE INTO followups (id,patient_id,patient_name,summary,due_at,status,created_at,updated_at) VALUES
 ('QC-0716-001','PT-001','样本批次 A001','质控曲线复核','2026-07-17','待完成','2026-07-16T00:00:00Z','2026-07-16T00:00:00Z'),
 ('QC-0716-002','PT-002','样本批次 A002','试剂批号复核','2026-07-17','待完成','2026-07-16T00:00:00Z','2026-07-16T00:00:00Z'),
 ('QC-0716-003','PT-003','样本批次 A003','检测结果双人复核','2026-07-18','待完成','2026-07-16T00:00:00Z','2026-07-16T00:00:00Z'),
 ('QC-0716-004','PT-004','样本批次 A004','设备校准记录归档','2026-07-18','待完成','2026-07-16T00:00:00Z','2026-07-16T00:00:00Z'),
 ('QC-0716-005','PT-005','样本批次 A005','异常结果复核','2026-07-19','待完成','2026-07-16T00:00:00Z','2026-07-16T00:00:00Z'),
 ('QC-0716-006','PT-006','样本批次 A006','质控品库存核对','2026-07-19','待完成','2026-07-16T00:00:00Z','2026-07-16T00:00:00Z'),
 ('QC-0716-007','PT-007','样本批次 A007','仪器运行日志归档','2026-07-20','待完成','2026-07-16T00:00:00Z','2026-07-16T00:00:00Z'),
 ('QC-0716-008','PT-008','样本批次 A008','复测原因记录','2026-07-20','待完成','2026-07-16T00:00:00Z','2026-07-16T00:00:00Z'),
 ('QC-0716-009','PT-009','样本批次 A009','报告发布前复核','2026-07-21','待完成','2026-07-16T00:00:00Z','2026-07-16T00:00:00Z'),
 ('QC-0716-010','PT-010','样本批次 A010','满意度与时效复盘','2026-07-21','待完成','2026-07-16T00:00:00Z','2026-07-16T00:00:00Z'),
 ('QC-0716-011','PT-011','样本批次 A011','设备维护提醒','2026-07-22','待完成','2026-07-16T00:00:00Z','2026-07-16T00:00:00Z'),
 ('QC-0716-012','PT-012','样本批次 A012','质控数据归档','2026-07-22','待完成','2026-07-16T00:00:00Z','2026-07-16T00:00:00Z');
INSERT IGNORE INTO samples (id,subject_alias,sample_type,collected_at,status,created_at,updated_at) VALUES
 ('SM-0717-001','受检者-001','血液','2026-07-17T08:00:00Z','待送检','2026-07-17T00:00:00Z','2026-07-17T00:00:00Z'),
 ('SM-0717-002','受检者-002','尿液','2026-07-17T09:00:00Z','已接收','2026-07-17T00:00:00Z','2026-07-17T00:10:00Z'),
 ('SM-0717-003','受检者-003','咽拭子','2026-07-17T09:30:00Z','检验中','2026-07-17T00:20:00Z','2026-07-17T01:00:00Z'),
 ('SM-0717-004','受检者-004','血液','2026-07-17T10:00:00Z','待复核','2026-07-17T00:30:00Z','2026-07-17T01:20:00Z'),
 ('SM-0717-005','受检者-005','尿液','2026-07-17T10:30:00Z','已出报告','2026-07-17T00:40:00Z','2026-07-17T01:30:00Z'),
 ('SM-0717-006','受检者-006','咽拭子','2026-07-17T11:00:00Z','已归档','2026-07-17T00:50:00Z','2026-07-17T01:40:00Z');
INSERT IGNORE INTO sample_tests (id,sample_id,name,status,created_at) VALUES
 ('SM-0717-001-T1','SM-0717-001','基础检验','待检验','2026-07-17T00:00:00Z'),
 ('SM-0717-002-T1','SM-0717-002','基础检验','待检验','2026-07-17T00:00:00Z'),
 ('SM-0717-003-T1','SM-0717-003','基础检验','待检验','2026-07-17T00:00:00Z'),
 ('SM-0717-004-T1','SM-0717-004','基础检验','待检验','2026-07-17T00:00:00Z'),
 ('SM-0717-005-T1','SM-0717-005','基础检验','待检验','2026-07-17T00:00:00Z'),
 ('SM-0717-006-T1','SM-0717-006','基础检验','待检验','2026-07-17T00:00:00Z');
INSERT IGNORE INTO sample_events (id,sample_id,action,from_status,to_status,actor,created_at) VALUES
 ('SM-0717-002-E1','SM-0717-002','接收样本','待送检','已接收','seed','2026-07-17T01:00:00Z'),
 ('SM-0717-003-E1','SM-0717-003','接收样本','待送检','已接收','seed','2026-07-17T01:00:00Z'),
 ('SM-0717-003-E2','SM-0717-003','开始检验','已接收','检验中','seed','2026-07-17T02:00:00Z'),
 ('SM-0717-004-E1','SM-0717-004','接收样本','待送检','已接收','seed','2026-07-17T01:00:00Z'),
 ('SM-0717-004-E2','SM-0717-004','开始检验','已接收','检验中','seed','2026-07-17T02:00:00Z'),
 ('SM-0717-004-E3','SM-0717-004','提交报告','检验中','待复核','seed','2026-07-17T03:00:00Z'),
 ('SM-0717-005-E1','SM-0717-005','接收样本','待送检','已接收','seed','2026-07-17T01:00:00Z'),
 ('SM-0717-005-E2','SM-0717-005','开始检验','已接收','检验中','seed','2026-07-17T02:00:00Z'),
 ('SM-0717-005-E3','SM-0717-005','提交报告','检验中','待复核','seed','2026-07-17T03:00:00Z'),
 ('SM-0717-005-E4','SM-0717-005','复核报告','待复核','已出报告','seed','2026-07-17T04:00:00Z'),
 ('SM-0717-006-E1','SM-0717-006','接收样本','待送检','已接收','seed','2026-07-17T01:00:00Z'),
 ('SM-0717-006-E2','SM-0717-006','开始检验','已接收','检验中','seed','2026-07-17T02:00:00Z'),
 ('SM-0717-006-E3','SM-0717-006','提交报告','检验中','待复核','seed','2026-07-17T03:00:00Z'),
 ('SM-0717-006-E4','SM-0717-006','复核报告','待复核','已出报告','seed','2026-07-17T04:00:00Z'),
 ('SM-0717-006-E5','SM-0717-006','归档报告','已出报告','已归档','seed','2026-07-17T05:00:00Z');
